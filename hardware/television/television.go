// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package television

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

// the number of synced frames where we can expect things to be in flux.
const leadingFrames = 1

// the number of synced frames required before the TV is considered to be
// stable. once the tv is stable then specification switching cannot happen.
const stabilityThreshold = 6

type vsync struct {
	active          bool
	activeClock     int
	activeScanlines int

	// the ideal scanline at which the "new frame" is triggered. this can be
	// thought of as the number of scanline between valid VSYNC signals. as
	// such, it is only reset on reception of a valid VSYNC signal
	scanline int

	// the scanline at which a "new frame" is actually triggered. this will be
	// different than the scanlines field during the synchronisation process causing
	// the screen to visually roll
	flybackScanline int

	// a vary good example of a ROM that requires correct handling of
	// natural flyback is Andrew Davies' Chess (3e+ test rom)
	//
	// (06/01/21) another example is the Artkaris NTSC version of Lili
}

func (v vsync) isSynced() bool {
	return v.scanline == v.flybackScanline
}

// State encapsulates the television values that can change from moment to
// moment. Used by the rewind system when recording the current television
// state.
type State struct {
	// the FrameInfo for the current frame
	frameInfo FrameInfo

	// the requested television specification from outside of the TV package.
	// this should only be set with the SetSpec() function
	reqSpecID string

	// state of the television. these values correspond to the most recent
	// signal received
	//
	// not using TelevisionCoords type here.
	//
	// clock field counts from zero not negative specification.ClksHblank
	frameNum int
	scanline int
	clock    int

	// the number of consistent frames seen after reset. once the count reaches
	// stabilityThreshold then Stable flag in the FrameInfo type is set to
	// true.
	//
	// once stableFrames reaches stabilityThreshold it is never reset except by
	// an explicit call to Reset() or by SetSpec() with the forced flag and a
	// requested spec of "AUTO"
	stableFrames int

	// record of signal attributes from the last call to Signal()
	lastSignal signal.SignalAttributes

	// vsync control
	vsync vsync

	// frame resizer
	resizer Resizer
}

func (s *State) String() string {
	// I would like to include the lastSignal string in this too but I'm
	// leaving it out for now because of existing video regression entries with
	// TV state will fail with it added.
	//
	// !!TODO: consider adding lastSignal information to TV state string.
	return fmt.Sprintf("FR=%04d SL=%03d CL=%03d", s.frameNum, s.scanline, s.clock-specification.ClksHBlank)
}

// Snapshot makes a copy of the television state.
func (s *State) Snapshot() *State {
	n := *s
	return &n
}

// spec string MUST be normalised with specification.NormaliseReqSpecID()
func (s *State) setSpec(spec string) {
	switch spec {
	case "AUTO":
		s.frameInfo = NewFrameInfo(specification.SpecNTSC)
		s.resizer.setSpec(specification.SpecNTSC)
		s.stableFrames = 0
	case "NTSC":
		s.frameInfo = NewFrameInfo(specification.SpecNTSC)
		s.resizer.setSpec(specification.SpecNTSC)
	case "PAL":
		s.frameInfo = NewFrameInfo(specification.SpecPAL)
		s.resizer.setSpec(specification.SpecPAL)
	case "PAL-M":
		s.frameInfo = NewFrameInfo(specification.SpecPAL_M)
		s.resizer.setSpec(specification.SpecPAL_M)
	case "SECAM":
		s.frameInfo = NewFrameInfo(specification.SpecSECAM)
		s.resizer.setSpec(specification.SpecSECAM)
	case "PAL60":
		// we treat PAL60 as just another name for PAL. whether it is 60HZ or
		// not depends on the generated frame
		s.frameInfo = NewFrameInfo(specification.SpecPAL)
		s.resizer.setSpec(specification.SpecPAL)
	}
}

// SetReqSpec sets the requested specification ID
func (s *State) SetReqSpec(spec string) {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return
	}
	s.reqSpecID = spec
	s.setSpec(spec)
}

// GetReqSpecID returns the specification that was most recently requested
func (s *State) GetReqSpecID() string {
	return s.reqSpecID
}

// GetLastSignal returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (s *State) GetLastSignal() signal.SignalAttributes {
	return s.lastSignal
}

// GetFrameInfo returns the television's current frame information.
func (s *State) GetFrameInfo() FrameInfo {
	return s.frameInfo
}

// GetCoords returns an instance of coords.TelevisionCoords.
func (s *State) GetCoords() coords.TelevisionCoords {
	return coords.TelevisionCoords{
		Frame:    s.frameNum,
		Scanline: s.scanline,
		Clock:    s.clock - specification.ClksHBlank,
	}
}

// Television is a Television implementation of the Television interface. In all
// honesty, it's most likely the only implementation required.
type Television struct {
	env *environment.Environment

	// the ID with which the television was created. this overrides all spec
	// changes unles the value is AUTO
	creationSpecID string

	// vcs will be nil unless AttachVCS() has been called
	vcs VCSReturnChannel

	// framerate limiter
	lmtr limiter

	// list of PixelRenderer implementations to consult
	renderers []PixelRenderer

	// list of FrameTrigger implementations to consult
	frameTriggers []FrameTrigger

	// list of ScanlineTrigger implementations to consult
	scanlineTriggers []ScanlineTrigger

	// list of audio mixers to consult
	mixers []AudioMixer

	// realtime mixer. only one allowed
	realtimeMixer RealtimeAudioMixer

	// instance of current state (as supported by the rewind system)
	state *State

	// signals are buffered before being forwarded to a PixelRenderer.
	//
	// signals in the array are always consecutive.
	//
	// the signals in the array will never cross a frame boundary. ie. all
	// signals belong to the same frame.
	//
	// the first signal in the array is not necessary at scanline zero, clock
	// zero
	//
	// information about which scanline/clock a SignalAttribute corresponds to
	// is part of the SignaalAttributes information (see signal package).
	//
	// because each SignalAttribute can be decoded for scanline and clock
	// information the array can be sliced freely
	signals []signal.SignalAttributes

	// the index of the most recent Signal()
	currentSignalIdx int

	// the index of the first Signal() in the frame
	firstSignalIdx int

	// copy of the signals and index fields from the previous frame. we use
	// solely to support the realtime audio mixer
	//
	// updated in renderSignals() function. might need more nuanced
	// copying/appending. for example if renderSignals() is called multiple
	// times per frame. currently this will only happen in the debugger when
	// execution is halted mid frame so I don't think it's an issue
	prevSignals       []signal.SignalAttributes
	prevSignalLastIdx int
	prevSignalFirst   int

	// state of emulation
	emulationState govern.State
}

// NewTelevision creates a new instance of the television type, satisfying the
// Television interface.
func NewTelevision(spec string) (*Television, error) {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return nil, fmt.Errorf("television: unsupported spec (%s)", spec)
	}

	tv := &Television{
		creationSpecID: spec,
		state: &State{
			reqSpecID: spec,
		},
		signals:     make([]signal.SignalAttributes, specification.AbsoluteMaxClks),
		prevSignals: make([]signal.SignalAttributes, specification.AbsoluteMaxClks),
	}

	// initialise frame rate limiter
	tv.lmtr.init(tv)
	tv.SetFPS(-1)

	// set specification
	tv.setSpec(spec)

	// empty list of renderers
	tv.renderers = make([]PixelRenderer, 0)

	return tv, nil
}

func (tv *Television) String() string {
	return tv.state.String()
}

// Reset the television to an initial state.
func (tv *Television) Reset(keepFrameNum bool) error {
	// we definitely do not call this on television initialisation because the
	// rest of the system may not be yet be in a suitable state

	// we're no longer resetting the TV spec on Reset(). doing so interferes
	// with the flexibility required to set the spec based on filename settings
	// etc.

	if !keepFrameNum {
		tv.state.frameNum = 0
	}

	tv.state.clock = 0
	tv.state.scanline = 0
	tv.state.stableFrames = 0
	tv.state.vsync.active = false
	tv.state.vsync.activeClock = 0
	tv.state.vsync.activeScanlines = 0
	tv.state.vsync.flybackScanline = specification.AbsoluteMaxScanlines
	tv.state.vsync.scanline = 0
	tv.state.lastSignal = signal.NoSignal

	for i := range tv.signals {
		tv.signals[i] = signal.NoSignal
	}
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0

	tv.setRefreshRate(tv.state.frameInfo.Spec.RefreshRate)
	tv.state.resizer.reset(tv.state.frameInfo.Spec)

	for _, r := range tv.renderers {
		r.Reset()
	}

	for _, m := range tv.mixers {
		m.Reset()
	}

	if tv.realtimeMixer != nil {
		tv.realtimeMixer.Reset()
	}

	return nil
}

// Snapshot makes a copy of the television state.
func (tv *Television) Snapshot() *State {
	return tv.state.Snapshot()
}

// Plumb attaches an existing television state.
func (tv *Television) Plumb(vcs VCSReturnChannel, state *State) {
	if state == nil {
		panic("television: cannot plumb in a nil state")
	}

	tv.state = state.Snapshot()

	// make sure vcs knows about current spec
	tv.vcs = vcs
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}

	// reset signal history
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0
}

// AttachVCS attaches an implementation of the VCSReturnChannel.
func (tv *Television) AttachVCS(env *environment.Environment, vcs VCSReturnChannel) {
	tv.env = env
	tv.vcs = vcs

	// notify the newly attached console of the current TV spec
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}
}

// AddPixelRenderer adds an implementation of PixelRenderer.
func (tv *Television) AddPixelRenderer(r PixelRenderer) {
	for i := range tv.renderers {
		if tv.renderers[i] == r {
			return
		}
	}
	tv.renderers = append(tv.renderers, r)
}

// RemovePixelRenderer removes a single PixelRenderer implementation from the
// list of renderers. Order is not maintained.
func (tv *Television) RemovePixelRenderer(r PixelRenderer) {
	for i := range tv.renderers {
		if tv.renderers[i] == r {
			tv.renderers[i] = tv.renderers[len(tv.renderers)-1]
			tv.renderers = tv.renderers[:len(tv.renderers)-1]
			return
		}
	}
}

// AddFrameTrigger adds an implementation of FrameTrigger.
func (tv *Television) AddFrameTrigger(f FrameTrigger) {
	for i := range tv.frameTriggers {
		if tv.frameTriggers[i] == f {
			return
		}
	}
	tv.frameTriggers = append(tv.frameTriggers, f)
}

// RemoveFrameTrigger removes a single FrameTrigger implementation from the
// list of triggers. Order is not maintained.
func (tv *Television) RemoveFrameTrigger(f FrameTrigger) {
	for i := range tv.frameTriggers {
		if tv.frameTriggers[i] == f {
			tv.frameTriggers[i] = tv.frameTriggers[len(tv.frameTriggers)-1]
			tv.frameTriggers = tv.frameTriggers[:len(tv.frameTriggers)-1]
			return
		}
	}
}

// AddScanlineTrigger adds an implementation of ScanlineTrigger.
func (tv *Television) AddScanlineTrigger(f ScanlineTrigger) {
	for i := range tv.scanlineTriggers {
		if tv.scanlineTriggers[i] == f {
			return
		}
	}
	tv.scanlineTriggers = append(tv.scanlineTriggers, f)
}

// RemoveScanlineTrigger removes a single ScanlineTrigger implementation from the
// list of triggers. Order is not maintained.
func (tv *Television) RemoveScanlineTrigger(f ScanlineTrigger) {
	for i := range tv.scanlineTriggers {
		if tv.scanlineTriggers[i] == f {
			tv.scanlineTriggers[i] = tv.scanlineTriggers[len(tv.scanlineTriggers)-1]
			tv.scanlineTriggers = tv.scanlineTriggers[:len(tv.scanlineTriggers)-1]
			return
		}
	}
}

// AddAudioMixer adds an implementation of AudioMixer.
func (tv *Television) AddAudioMixer(m AudioMixer) {
	for i := range tv.mixers {
		if tv.mixers[i] == m {
			return
		}
	}
	tv.mixers = append(tv.mixers, m)
}

// RemoveAudioMixer removes a single AudioMixer implementation from the
// list of mixers. Order is not maintained.
func (tv *Television) RemoveAudioMixer(m AudioMixer) {
	for i := range tv.mixers {
		if tv.mixers[i] == m {
			tv.mixers[i] = tv.mixers[len(tv.mixers)-1]
			tv.mixers = tv.mixers[:len(tv.mixers)-1]
			return
		}
	}
}

// AddRealtimeAudioMixer adds a RealtimeAudioMixer. Any previous assignment is
// lost.
func (tv *Television) AddRealtimeAudioMixer(m RealtimeAudioMixer) {
	tv.realtimeMixer = m
}

// RemoveRealtimeAudioMixer removes any RealtimeAudioMixer implementation from
// the Television.
func (tv *Television) RemoveRealtimeAudioMixer() {
	tv.realtimeMixer = nil
}

// some televisions may need to conclude and/or dispose of resources
// gently. implementations of End() should call EndRendering() and
// EndMixing() on each PixelRenderer and AudioMixer that has been added.
//
// for simplicity, the Television should be considered unusable
// after EndRendering() has been called.
func (tv *Television) End() error {
	var err error

	// call new frame for all renderers
	for _, r := range tv.renderers {
		err = r.EndRendering()
	}

	// flush audio for all mixers
	for _, m := range tv.mixers {
		err = m.EndMixing()
	}

	return err
}

// Signal updates the current state of the television.
func (tv *Television) Signal(sig signal.SignalAttributes) {
	// examine signal for resizing possibility.
	tv.state.resizer.examine(tv.state, sig)

	// count VSYNC scanlines
	if tv.state.vsync.active && tv.state.clock == tv.state.vsync.activeClock {
		tv.state.vsync.activeScanlines++
	}

	// check for change of VSYNC signal
	if sig&signal.VSync != tv.state.lastSignal&signal.VSync {
		if sig&signal.VSync == signal.VSync {
			// VSYNC has started
			tv.state.vsync.active = true
			tv.state.vsync.activeScanlines = 0
			tv.state.vsync.activeClock = tv.state.clock
		} else {
			// the number of scanlines that the VSYNC has been held should be
			// somewhere between two and three. anything over three is fine
			// TODO: make this a user preference
			if tv.state.vsync.activeScanlines >= tv.env.Prefs.TV.VSYNCscanlines.Get().(int) {
				recovery := max(preferences.VSYNCrecoveryMin,
					min(preferences.VSYNCrecoveryMax,
						tv.env.Prefs.TV.VSYNCrecovery.Get().(int)))

				// adjust flyback scanline until it matches the vsync scanline.
				// also adjust the actual scanline if it's not zero
				if tv.state.vsync.scanline < tv.state.vsync.flybackScanline {
					adj := ((tv.state.vsync.flybackScanline - tv.state.vsync.scanline) * recovery) / 100
					tv.state.vsync.flybackScanline = tv.state.vsync.scanline + adj
					tv.state.scanline = (tv.state.scanline * recovery) / 100
				} else if tv.state.vsync.scanline > tv.state.vsync.flybackScanline {
					// if the current frame was created from a synchronised position then simply
					// move the flyback scanline to the new value. there is also a user preference
					// that controls whether to allow this or not
					if tv.state.frameInfo.IsSynced[0] && tv.state.frameInfo.IsSynced[1] && !tv.env.Prefs.TV.VSYNCimmediateDesync.Get().(bool) {
						tv.state.vsync.flybackScanline = tv.state.vsync.scanline % specification.AbsoluteMaxScanlines
					} else {
						tv.state.vsync.flybackScanline = specification.AbsoluteMaxScanlines
					}
				} else if tv.state.vsync.scanline == tv.state.vsync.flybackScanline {
					// continue to adjust scanline until it is zero
					if tv.state.scanline > 0 {
						tv.state.scanline = (tv.state.scanline * recovery) / 100
					}
				}

				// reset VSYNC scanline count only when we've received a valid VSYNC signal
				tv.state.vsync.scanline = 0

			} else {
				// set flyback scanline to the maximum if we receive an invalid VSYNC signal
				tv.state.vsync.flybackScanline = specification.AbsoluteMaxScanlines
			}

			// end of VSYNC
			tv.state.vsync.active = false
		}
	}

	// a Signal() is by definition a new color clock. increase the horizontal count
	tv.state.clock++

	// once we reach the scanline's back-porch we'll reset the clock counter
	// and wait for the HSYNC signal. we do this so that the front-porch and
	// back-porch are 'together' at the beginning of the scanline. this isn't
	// strictly technically correct but it's convenient to think about
	// scanlines in this way (rather than having a split front and back porch)
	if tv.state.clock >= specification.ClksScanline {
		tv.state.clock = 0

		// bump scanline counter
		tv.state.scanline++
		tv.state.vsync.scanline++

		if tv.state.scanline >= tv.state.vsync.flybackScanline {
			err := tv.newFrame()
			if err != nil {
				logger.Log(tv.env, "TV", err.Error())
			}
		} else {
			// if we're not at end of screen then indicate new scanline
			err := tv.newScanline()
			if err != nil {
				logger.Log(tv.env, "TV", err.Error())
			}
		}
	}

	// we've "faked" the flyback signal above when clock reached
	// horizClksScanline. we need to handle the real flyback signal however, by
	// making sure we're at the correct clock value.
	//
	// this should be seen as a special condition and one that could be
	// removed if the TV signal was emulated properly. for now the range check
	// is to enable the RSYNC smooth scrolling trick to be displayed correctly.
	//
	// https://atariage.com/forums/topic/224946-smooth-scrolling-playfield-i-think-ive-done-it
	if sig&signal.HSync == signal.HSync && tv.state.lastSignal&signal.HSync != signal.HSync {
		if tv.state.clock < 13 || tv.state.clock > 22 {
			tv.state.clock = 16
		}
	}

	// doing nothing with CBURST signal

	// assume that clock and scanline are constrained elsewhere such that the
	// index can never run past the end of the signals array
	tv.currentSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	// sometimes the current signal can come out "behind" the firstSignalIdx.
	// this can happen when RSYNC is triggered on the first scanline of the
	// frame. not common but we should handle it
	//
	// in practical terms, if we don't handle this then sending signals to the
	// audio mixers will cause a "slice bounds out of range" panic
	if tv.currentSignalIdx < tv.firstSignalIdx {
		tv.firstSignalIdx = tv.currentSignalIdx
	}

	// augment television signal before storing and sending to pixel renderers
	sig &= ^signal.Index
	sig |= signal.SignalAttributes(tv.currentSignalIdx << signal.IndexShift)

	// write the signal into the correct index of the signals array.

	tv.signals[tv.currentSignalIdx] = sig

	// record the current signal settings so they can be used for reference
	// during the next call to Signal()
	tv.state.lastSignal = sig

	// record signal history
	if tv.currentSignalIdx >= len(tv.signals) {
		err := tv.renderSignals()
		if err != nil {
			logger.Log(tv.env, "TV", err.Error())
		}
	}
}

func (tv *Television) newScanline() error {
	// notify renderers of new scanline
	for _, r := range tv.renderers {
		err := r.NewScanline(tv.state.scanline)
		if err != nil {
			return err
		}
	}

	// check for realtime mixing requirements. if it is required then
	// immediately push the audio data from the previous frame to the mixer
	if tv.realtimeMixer != nil && tv.emulationState == govern.Running && tv.state.frameInfo.Stable {
		if tv.realtimeMixer.MoreAudio() {
			err := tv.realtimeMixer.SetAudio(tv.prevSignals[:tv.prevSignalLastIdx])
			if err != nil {
				return err
			}
		}
	}

	// process all ScanlineTriggers
	for _, r := range tv.scanlineTriggers {
		err := r.NewScanline(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	tv.lmtr.checkScanline()

	return nil
}

func (tv *Television) newFrame() error {
	// increase or reset stable frame count as required
	if tv.state.stableFrames <= stabilityThreshold {
		if tv.state.vsync.isSynced() {
			tv.state.stableFrames++
			tv.state.frameInfo.Stable = tv.state.stableFrames >= stabilityThreshold
		} else {
			tv.state.stableFrames = 0
			tv.state.frameInfo.Stable = false
		}
	}

	// specification change between NTSC and PAL. PAL-M is treated the same as
	// NTSC in this instance
	//
	// Note that setSpec() resets the frameInfo completely so we must set the
	// framenumber and vsynced after any possible setSpec()
	if tv.state.stableFrames > leadingFrames && tv.state.stableFrames < stabilityThreshold {
		switch tv.state.frameInfo.Spec.ID {
		case specification.SpecPAL_M.ID:
			fallthrough
		case specification.SpecNTSC.ID:
			if tv.state.reqSpecID == "AUTO" && tv.state.scanline > specification.PALTrigger {
				tv.setSpec("PAL")
			}
		case specification.SpecPAL.ID:
			if tv.state.reqSpecID == "AUTO" && tv.state.scanline <= specification.PALTrigger {
				tv.setSpec("NTSC")
			}
		}
	}

	// update frame number
	tv.state.frameInfo.FrameNum = tv.state.frameNum

	// note whether newFrame() was the result of a valid VSYNC or a natural flyback
	tv.state.frameInfo.IsSynced[1] = tv.state.frameInfo.IsSynced[0]
	tv.state.frameInfo.IsSynced[0] = tv.state.vsync.isSynced()

	// commit any resizing that maybe pending
	err := tv.state.resizer.commit(tv.state)
	if err != nil {
		return err
	}

	// record total scanlines and refresh rate if changed. note that this is
	// independent of the resizer.commit() call above.
	//
	// this is important to do and failure to set the refresh rate correctly
	// is most noticeable in the Supercharger tape loading process
	if tv.state.frameInfo.TotalScanlines != tv.state.scanline {
		tv.state.frameInfo.TotalScanlines = tv.state.scanline
		tv.state.frameInfo.RefreshRate = tv.state.frameInfo.Spec.HorizontalScanRate / float32(tv.state.scanline)
		tv.setRefreshRate(tv.state.frameInfo.RefreshRate)
		tv.state.frameInfo.Jitter = true
	} else {
		tv.state.frameInfo.Jitter = false
	}

	// prepare for next frame
	tv.state.frameNum++
	tv.state.scanline = 0

	// nullify unused signals at end of frame
	for i := tv.currentSignalIdx; i < len(tv.signals); i++ {
		tv.signals[i] = signal.NoSignal
	}

	// set pending pixels
	err = tv.renderSignals()
	if err != nil {
		return err
	}

	// process all pixel renderers
	for _, r := range tv.renderers {
		err := r.NewFrame(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	// process all FrameTriggers
	for _, r := range tv.frameTriggers {
		err := r.NewFrame(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	// check frame rate
	tv.lmtr.checkFrame()

	// measure frame rate
	tv.lmtr.measureActual()

	// signal index at beginning of new frame
	tv.firstSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	return nil
}

// renderSignals forwards pixels in the signalHistory buffer to all pixel
// renderers and audio mixers.
func (tv *Television) renderSignals() error {
	// do not render pixels if emulation is in the rewinding state
	if tv.emulationState != govern.Rewinding {
		for _, r := range tv.renderers {
			err := r.SetPixels(tv.signals, tv.currentSignalIdx)
			if err != nil {
				return fmt.Errorf("television: %w", err)
			}
		}
	}

	// ... but we do mix audio even if the emulation is rewinding

	// update realtime mixers
	//
	// an additional condition saying the realtimeMixer is used only once the
	// frame is stable has been removed. it was thought to improve sound on
	// startup for some ROMs but in some pathological cases it means sound is
	// never output. in particular, the tunabit demo ROM.
	//
	// https://atariage.com/forums/topic/274172-tiatune-tia-music-player-with-correct-tuning/
	if tv.realtimeMixer != nil {
		err := tv.realtimeMixer.SetAudio(tv.signals[tv.firstSignalIdx:tv.currentSignalIdx])
		if err != nil {
			return fmt.Errorf("television: %w", err)
		}
	}

	// update regular mixers
	for _, m := range tv.mixers {
		err := m.SetAudio(tv.signals[tv.firstSignalIdx:tv.currentSignalIdx])
		if err != nil {
			return fmt.Errorf("television: %w", err)
		}
	}

	// make a copy of signals just rendered
	copy(tv.prevSignals, tv.signals)
	tv.prevSignalLastIdx = tv.currentSignalIdx
	tv.prevSignalFirst = tv.firstSignalIdx

	return nil
}

// SetSpec sets the television's specification if the creation ID is AUTO. This
// means that the television specification on creation overrides all other
// specifcation requests
//
// The forced argument overrides this rule.
func (tv *Television) SetSpec(spec string, forced bool) error {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return fmt.Errorf("television: unsupported spec (%s)", spec)
	}

	if forced {
		tv.creationSpecID = spec
	} else if tv.creationSpecID != "AUTO" {
		return nil
	}

	tv.state.reqSpecID = spec
	tv.setSpec(spec)

	return nil
}

func (tv *Television) setSpec(spec string) {
	tv.state.setSpec(spec)
	tv.setRefreshRate(tv.state.frameInfo.Spec.RefreshRate)
}

// setRefreshRate of TV. calls frame limiter and pixel renderers as appropriate
func (tv *Television) setRefreshRate(rate float32) {
	tv.lmtr.setRefreshRate(rate)
	tv.lmtr.setRate(rate)

	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}
}

// SetEmulationState is called by emulation whenever state changes. How we
// handle incoming signals depends on the current state.
func (tv *Television) SetEmulationState(state govern.State) error {
	prev := tv.emulationState
	tv.emulationState = state

	switch prev {
	case govern.Paused:
		// start off the unpaused state by measuring the current framerate.
		// this "clears" the ticker channel and means the feedback from
		// GetActualFPS() is less misleading
		tv.lmtr.measureActual()

	case govern.Rewinding:
		tv.renderSignals()
	}

	switch state {
	case govern.Paused:
		err := tv.renderSignals()
		if err != nil {
			return err
		}
	}

	return nil
}

// NudgeFPSCap stops the FPS limiter for the specified number of frames. A value
// of zero (or less) will stop any existing nudge
func (tv *Television) NudgeFPSCap(frames int) {
	if frames < 0 {
		frames = 0
	}
	tv.lmtr.nudge.Store(int32(frames))
}

// SetFPSCap whether the emulation should wait for FPS limiter. Returns the
// setting as it was previously.
func (tv *Television) SetFPSCap(limit bool) bool {
	prev := tv.lmtr.active
	tv.lmtr.active = limit

	// notify all pixel renderers that are interested in the FPS cap
	for i := range tv.renderers {
		if r, ok := tv.renderers[i].(PixelRendererFPSCap); ok {
			r.SetFPSCap(limit)
		}
	}

	return prev
}

// SetFPS requests the number frames per second. This overrides the frame rate of
// the specification. A negative value restores frame rate to the ideal value
// (the frequency of the incoming signal).
func (tv *Television) SetFPS(fps float32) {
	tv.lmtr.setRate(fps)
}

// GetReqFPS returns the requested number of frames per second. Compare with
// GetActualFPS() to check for accuracy.
//
// IS goroutine safe.
func (tv *Television) GetReqFPS() float32 {
	return tv.lmtr.requested.Load().(float32)
}

// GetActualFPS returns the current number of frames per second and the
// detected frequency of the TV signal.
//
// Note that FPS measurement still works even when frame capping is disabled.
//
// IS goroutine safe.
func (tv *Television) GetActualFPS() (float32, float32) {
	return tv.lmtr.measured.Load().(float32), tv.lmtr.refreshRate.Load().(float32)
}

// GetCreationSpecID returns the specification that was requested on creation.
func (tv *Television) GetCreationSpecID() string {
	return tv.creationSpecID
}

// GetReqSpecID returns the specification that was most recently requested.
func (tv *Television) GetReqSpecID() string {
	return tv.state.reqSpecID
}

// GetSpecID returns the current specification.
func (tv *Television) GetSpecID() string {
	return tv.state.frameInfo.Spec.ID
}

// GetFrameInfo returns the television's current frame information.
func (tv *Television) GetFrameInfo() FrameInfo {
	return tv.state.frameInfo
}

// GetLastSignal returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (tv *Television) GetLastSignal() signal.SignalAttributes {
	return tv.state.lastSignal
}

// GetCoords returns an instance of coords.TelevisionCoords.
//
// Like all Television functions this function is not safe to call from
// goroutines other than the one that created the Television.
func (tv *Television) GetCoords() coords.TelevisionCoords {
	return tv.state.GetCoords()
}

// SetRotation instructs the television to a different orientation. In truth,
// the television just forwards the request to the pixel renderers.
func (tv *Television) SetRotation(rotation specification.Rotation) {
	for _, r := range tv.renderers {
		if s, ok := r.(PixelRendererRotation); ok {
			s.SetRotation(rotation)
		}
	}
}

// GetResizer returns a copy of the television resizer in it's current state.
func (tv *Television) GetResizer() Resizer {
	return tv.state.resizer
}

// SetResizer sets the state of the television resizer and sets the current
// frame info accordingly. The validFrom value is the frame number at which the
// resize information was taken from
//
// Note that the Resizer type does not include specification information. When
// transferring state between television instances it is okay to call SetSpec()
// but it should be done before SetResizer() is called
func (tv *Television) SetResizer(rz Resizer, validFrom int) {
	tv.state.resizer = rz
	tv.state.resizer.validFrom = validFrom
	if tv.state.resizer.usingVBLANK {
		tv.state.frameInfo.VisibleTop = tv.state.resizer.vblankTop
		tv.state.frameInfo.VisibleBottom = tv.state.resizer.vblankBottom
	} else {
		tv.state.frameInfo.VisibleTop = tv.state.resizer.blackTop
		tv.state.frameInfo.VisibleBottom = tv.state.resizer.blackBottom
	}

	// call new frame for all pixel renderers. this will force the new size
	// information to be handled immediately but it won't result in a "phantom"
	// frame because we won't change the FrameNum field in the FrameInfo
	for _, r := range tv.renderers {
		r.NewFrame(tv.state.frameInfo)
	}
}
