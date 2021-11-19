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

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/userinput"
)

func (dbg *Debugger) userInputHandler_catchUpLoop() {
	select {
	case ev := <-dbg.events.UserInput:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			dbg.rewindMouseWheelAccumulation += int(ev.Delta)
		default:
			dbg.events.UserInput <- ev
		}
	default:
	}
}

func (dbg *Debugger) userInputHandler(ev userinput.Event) error {
	// quite event
	switch ev.(type) {
	case userinput.EventQuit:
		dbg.running = false
		return curated.Errorf(terminal.UserQuit)
	}

	// mode specific special input (not passed to the VCS as controller input)
	switch dbg.Mode() {
	case emulation.ModePlay:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			amount := int(ev.Delta) + dbg.rewindMouseWheelAccumulation
			dbg.rewindMouseWheelAccumulation = 0
			dbg.RewindByAmount(amount)
			return nil

		case userinput.EventKeyboard:
			if ev.Down {
				switch ev.Key {
				case "Left":
					if ev.Mod == userinput.KeyModShift {
						if dbg.rewindKeyboardAccumulation >= 0 {
							dbg.rewindKeyboardAccumulation = -1
						}
						return nil
					}
				case "Right":
					if ev.Mod == userinput.KeyModShift {
						if dbg.rewindKeyboardAccumulation <= 0 {
							dbg.rewindKeyboardAccumulation = 1
						}
						return nil
					}
				}
			} else {
				dbg.rewindKeyboardAccumulation = 0
				if dbg.State() != emulation.Running {
					return nil
				}
			}
		case userinput.EventGamepadButton:
			if ev.Down {
				switch ev.Button {
				case userinput.GamepadButtonBumperLeft:
					if dbg.rewindKeyboardAccumulation >= 0 {
						dbg.rewindKeyboardAccumulation = -1
					}
					return nil
				case userinput.GamepadButtonBumperRight:
					if dbg.rewindKeyboardAccumulation <= 0 {
						dbg.rewindKeyboardAccumulation = 1
					}
					return nil
				}
			} else {
				dbg.rewindKeyboardAccumulation = 0
				if dbg.State() != emulation.Running {
					return nil
				}
			}
		}
	}

	// applies to both playmode and debugger
	switch ev := ev.(type) {
	case userinput.EventGamepadButton:
		if ev.Down {
			switch ev.Button {
			case userinput.GamepadButtonBack:
				if dbg.State() != emulation.Paused {
					dbg.SetFeature(emulation.ReqSetPause, true)
				} else {
					dbg.SetFeature(emulation.ReqSetPause, false)
				}
				return nil
			case userinput.GamepadButtonGuide:
				switch dbg.Mode() {
				case emulation.ModePlay:
					dbg.SetFeature(emulation.ReqSetMode, emulation.ModeDebugger)
				case emulation.ModeDebugger:
					dbg.SetFeature(emulation.ReqSetMode, emulation.ModePlay)
				}
			}
		}
	}

	// pass to VCS controller emulation via the userinput package
	handled, err := dbg.controllers.HandleUserInput(ev, dbg.vcs.RIOT.Ports)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// the user input was something that controls the emulation (eg. a joystick
	// direction). unpause if the emulation is currently paused
	//
	// * we're only allowing this for playmode
	if dbg.Mode() == emulation.ModePlay && dbg.State() == emulation.Paused && handled {
		dbg.SetFeature(emulation.ReqSetPause, false)
	}

	return nil
}
