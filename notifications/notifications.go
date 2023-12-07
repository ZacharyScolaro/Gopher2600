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

package notifications

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"

// Notify describes events that somehow change the presentation of the
// emulation. These notifications can be used to present additional information
// to the user
type Notify string

// List of defined notifications.
const (
	// emulation events
	NotifyInitialising  Notify = "NotifyInitialising"
	NotifyPause         Notify = "NotifyPause"
	NotifyRun           Notify = "NotifyRun"
	NotifyRewindBack    Notify = "NotifyRewindBack"
	NotifyRewindFoward  Notify = "NotifyRewindFoward"
	NotifyRewindAtStart Notify = "NotifyRewindAtStart"
	NotifyRewindAtEnd   Notify = "NotifyRewindAtEnd"
	NotifyScreenshot    Notify = "NotifyScreenshot"
	NotifyMute          Notify = "NotifyMute"
	NotifyUnmute        Notify = "NotifyUnmute"

	// the following notifications relate to events generated by a cartridge

	// LoadStarted is raised for Supercharger mapper whenever a new tape read
	// sequence if started.
	NotifySuperchargerLoadStarted Notify = "NotifySuperchargerLoadStarted"

	// If Supercharger is loading from a fastload binary then this event is
	// raised when the loading has been completed.
	NotifySuperchargerFastloadEnded Notify = "NotifySuperchargerFastloadEnded"

	// If Supercharger is loading from a sound file (eg. mp3 file) then these
	// events area raised when the loading has started/completed.
	NotifySuperchargerSoundloadStarted Notify = "NotifySuperchargerSoundloadStarted"
	NotifySuperchargerSoundloadEnded   Notify = "NotifySuperchargerSoundloadEnded"

	// tape is rewinding.
	NotifySuperchargerSoundloadRewind Notify = "NotifySuperchargerSoundloadRewind"

	// PlusROM cartridge has been inserted.
	NotifyPlusROMInserted Notify = "NotifyPlusROMInserted"

	// PlusROM network activity.
	NotifyPlusROMNetwork Notify = "NotifyPlusROMNetwork"

	// PlusROM new installation
	NotifyPlusROMNewInstallation Notify = "NotifyPlusROMNewInstallation"

	// Moviecart started
	NotifyMovieCartStarted Notify = "NotifyMoveCartStarted"

	// unsupported DWARF data
	NotifyUnsupportedDWARF Notify = "NotifyUnsupportedDWARF"

	// coprocessor development information has been loaded
	NotifyCoprocDevStarted Notify = "NotifyCoprocDevStarted"
	NotifyCoprocDevEnded   Notify = "NotifyCoprocDevEnded"
)

// NotificationHook is used for direct communication between a the hardware and
// the emulation package. Not often used but necessary for (currently):
//
//	. Supercharger (eg. tape start/end)
//	. PlusROM (eg. new installation)
//
// The emulation understands how to interpret the event and forward the
// notification to the GUI using the gui.FeatureReq mechanism.
type NotificationHook func(cart mapper.CartMapper, notice Notify, args ...interface{}) error
