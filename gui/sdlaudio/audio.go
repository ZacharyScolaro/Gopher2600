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

package sdlaudio

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
	"github.com/jetsetilly/gopher2600/logger"

	"github.com/veandco/go-sdl2/sdl"
)

// number of samples in the SDL audio buffer at any one time. tha value is an
// estimate calculated by dividing the VCS audio frequency by the frame rate of
// the PAL specification.
//
//	31403 / 50 = 628.06
//
// (any value we use must be a multiple of four)
const bufferLength = 628

// realtimeDemand is the minimum number of samples that need to be in the SDL
// queue for MoreAudio() to return false
//
// no reasoning behind this value, it's just what seems to work well in practice
const realtimeDemand = bufferLength * 2

// if queued audio ever exceeds this value then push the audio into the SDL buffer
const maxQueueLength = 16384

// Audio outputs sound using SDL.
type Audio struct {
	id   sdl.AudioDeviceID
	spec sdl.AudioSpec

	buffer   []uint8
	bufferCt int

	Prefs *Preferences

	// local copy of some oft used preference values (too expensive to access
	// for the preferences system every time SetAudio() is called) . we'll
	// update these on every call to queueAudio()
	stereo     bool
	separation int
}

// NewAudio is the preferred method of initialisation for the Audio Type.
func NewAudio() (*Audio, error) {
	aud := &Audio{}

	var err error

	aud.Prefs, err = NewPreferences()
	if err != nil {
		return nil, curated.Errorf("sdlaudio: %v", err)
	}
	aud.stereo = aud.Prefs.Stereo.Get().(bool)

	spec := &sdl.AudioSpec{
		Freq:     audio.SampleFreq,
		Format:   sdl.AUDIO_S16MSB,
		Channels: 2,
		Samples:  uint16(bufferLength),
	}

	var actualSpec sdl.AudioSpec

	aud.id, err = sdl.OpenAudioDevice("", false, spec, &actualSpec, 0)
	if err != nil {
		return nil, curated.Errorf("sdlaudio: %v", err)
	}

	aud.spec = actualSpec

	logger.Logf("sdl: audio", "frequency: %d samples/sec", aud.spec.Freq)
	logger.Logf("sdl: audio", "format: %d", aud.spec.Format)
	logger.Logf("sdl: audio", "channels: %d", aud.spec.Channels)
	logger.Logf("sdl: audio", "buffer size: %d samples", bufferLength)
	logger.Logf("sdl: audio", "realtime demand: %d samples", realtimeDemand)
	logger.Logf("sdl: audio", "max samples: %d samples", maxQueueLength)
	logger.Logf("sdl: audio", "silence: %d", aud.spec.Silence)

	sdl.PauseAudioDevice(aud.id, false)

	aud.Reset()

	return aud, nil
}

// SetAudio implements the protocol.RealtimeAudioMixer interface.
func (aud *Audio) MoreAudio() bool {
	return sdl.GetQueuedAudioSize(aud.id) < realtimeDemand
}

// SetAudio implements the protocol.AudioMixer interface.
func (aud *Audio) SetAudio(sig []signal.SignalAttributes) error {
	for _, s := range sig {
		if s&signal.AudioUpdate != signal.AudioUpdate {
			continue
		}

		v0 := uint8((s & signal.AudioChannel0) >> signal.AudioChannel0Shift)
		v1 := uint8((s & signal.AudioChannel1) >> signal.AudioChannel1Shift)

		if aud.stereo {
			s0, s1 := mix.Stereo(v0, v1, aud.separation)
			aud.buffer[aud.bufferCt] = uint8(s0>>8) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(s0) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(s1>>8) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(s1) + aud.spec.Silence
			aud.bufferCt++
		} else {
			m := mix.Mono(v0, v1)
			aud.buffer[aud.bufferCt] = uint8(m>>8) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(m) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(m>>8) + aud.spec.Silence
			aud.bufferCt++
			aud.buffer[aud.bufferCt] = uint8(m) + aud.spec.Silence
			aud.bufferCt++
		}

		remaining := int(sdl.GetQueuedAudioSize(aud.id))
		if aud.bufferCt >= len(aud.buffer) {
			if err := aud.queueBuffer(); err != nil {
				return curated.Errorf("sdlaudio", err)
			}
		} else if remaining < realtimeDemand {
			if err := aud.queueBuffer(); err != nil {
				return curated.Errorf("sdlaudio", err)
			}
		} else if remaining > maxQueueLength {
			// if length of sdl: audio: queue is getting too long then clear it
			//
			// condition valid when the frame rate is SIGNIFICANTLY MORE than 50/60fps
			//
			// if we don't do this the video will get ahead of the audio (ie. the audio
			// will lag)
			//
			// this is a brute force approach but it'll do for now
			sdl.ClearQueuedAudio(aud.id)
			break // for loop
		}
	}

	return nil
}

func (aud *Audio) queueBuffer() error {
	err := sdl.QueueAudio(aud.id, aud.buffer[:aud.bufferCt])
	if err != nil {
		return err
	}
	aud.bufferCt = 0

	// update local preference values
	aud.stereo = aud.Prefs.Stereo.Get().(bool)
	aud.separation = aud.Prefs.Separation.Get().(int)

	return nil
}

// EndMixing implements the protocol.AudioMixer interface.
func (aud *Audio) EndMixing() error {
	sdl.CloseAudioDevice(aud.id)
	return nil
}

// Reset implements the protocol.AudioMixer interface.
func (aud *Audio) Reset() {
	aud.buffer = make([]uint8, bufferLength)
	aud.bufferCt = 0

	// fill buffers with silence
	for i := range aud.buffer {
		aud.buffer[i] = aud.spec.Silence
	}

	sdl.ClearQueuedAudio(aud.id)
}

// Mute silences the audio device.
func (aud *Audio) Mute(muted bool) {
	sdl.PauseAudioDevice(aud.id, muted)
}
