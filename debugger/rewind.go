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

package debugger

// support functions for the rewind package that require more knowledge of
// the debugger than would otherwise be available.

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// RewindByAmount moves forwards or backwards by specified frames. Negative
// numbers indicate backwards
func (dbg *Debugger) RewindByAmount(amount int) {
	switch dbg.Mode() {
	case govern.ModePlay:
		coords := dbg.vcs.TV.GetCoords()
		tl := dbg.Rewind.GetTimeline()

		if amount < 0 {
			if coords.Frame-1 < tl.AvailableStart {
				dbg.setState(govern.Paused, govern.PausedAtStart)
				return
			}
			dbg.setState(govern.Rewinding, govern.RewindingBackwards)
		}

		if amount > 0 {
			if coords.Frame+1 > tl.AvailableEnd {
				dbg.setState(govern.Paused, govern.PausedAtEnd)
				return
			}
			dbg.setState(govern.Rewinding, govern.RewindingForwards)
		}

		dbg.Rewind.GotoFrame(coords.Frame + amount)
		dbg.setState(govern.Paused, govern.Normal)

		return
	}

	panic(fmt.Sprintf("Rewind: unsupported mode (%v)", dbg.Mode()))
}

// RewindToFrame measure from the current frame.
func (dbg *Debugger) RewindToFrame(fn int, last bool) bool {
	switch dbg.Mode() {
	case govern.ModeDebugger:
		if dbg.State() == govern.Rewinding {
			return false
		}

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			// upate catchup context before starting rewind process
			dbg.catchupContext = catchupRewindToFrame

			if last {
				err := dbg.Rewind.GotoLast()
				if err != nil {
					return err
				}
			} else {
				err := dbg.Rewind.GotoFrame(fn)
				if err != nil {
					return err
				}
			}

			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushFunctionImmediate(func() {
			// set state to govern.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)
			if fn > dbg.vcs.TV.GetCoords().Frame {
				dbg.setState(govern.Rewinding, govern.RewindingForwards)
			} else {
				dbg.setState(govern.Rewinding, govern.RewindingBackwards)
			}
			dbg.unwindLoop(doRewind)
		})

		return true
	}

	panic(fmt.Sprintf("RewindToFrame: unsupported mode (%v)", dbg.Mode()))
}

// GotoCoords rewinds the emulation to the specified coordinates.
func (dbg *Debugger) GotoCoords(toCoords coords.TelevisionCoords) bool {
	switch dbg.Mode() {
	case govern.ModeDebugger:
		if dbg.State() == govern.Rewinding {
			return false
		}

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			// upate catchup context before starting rewind process
			dbg.catchupContext = catchupGotoCoords

			err := dbg.Rewind.GotoCoords(toCoords)
			if err != nil {
				return err
			}

			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushFunctionImmediate(func() {
			// set state to govern.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)

			fromCoords := dbg.vcs.TV.GetCoords()
			if coords.GreaterThan(toCoords, fromCoords) {
				dbg.setState(govern.Rewinding, govern.RewindingForwards)
			} else {
				dbg.setState(govern.Rewinding, govern.RewindingBackwards)
			}
			dbg.unwindLoop(doRewind)
		})

		return true
	}

	panic(fmt.Sprintf("GotoCoords: unsupported mode (%v)", dbg.Mode()))
}

// RerunLastNFrames measured from the current frame.
func (dbg *Debugger) RerunLastNFrames(frames int) bool {
	switch dbg.Mode() {
	case govern.ModeDebugger:
		if dbg.State() == govern.Rewinding {
			return false
		}

		// the disadvantage of RerunLastNFrames() is that it will always land on a
		// CPU instruction boundary (this is because we must unwind the existing
		// input loop before calling the rewind function)
		//
		// if we're in between instruction boundaries therefore we need to push a
		// GotoCoords() request. get the current coordinates now
		correctCoords := !dbg.liveDisasmEntry.Result.Final
		toCoords := dbg.vcs.TV.GetCoords()

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			err := dbg.Rewind.RerunLastNFrames(frames)
			if err != nil {
				return err
			}

			if correctCoords {
				err = dbg.Rewind.GotoCoords(toCoords)
				if err != nil {
					return err
				}
			}

			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushFunctionImmediate(func() {
			// upate catchup context before starting rewind process
			dbg.catchupContext = catrupRerunLastNFrames

			// set state to govern.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)
			dbg.setState(govern.Rewinding, govern.Normal)
			dbg.unwindLoop(doRewind)
		})

		return true
	}

	panic(fmt.Sprintf("RerunLastNFrames: unsupported mode (%v)", dbg.Mode()))
}
