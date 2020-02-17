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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v2"
)

const winRIOTTitle = "RIOT"

type winRIOT struct {
	windowManagement
	img *SdlImgui
}

func newWinRIOT(img *SdlImgui) (managedWindow, error) {
	win := &winRIOT{
		img: img,
	}

	return win, nil
}

func (win *winRIOT) destroy() {
}

func (win *winRIOT) id() string {
	return winRIOTTitle
}

// draw is called by service loop
func (win *winRIOT) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{359, 664}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{464, 48}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winRIOTTitle, &win.open, 0)
	imgui.Text(win.img.vcs.RIOT.String())
	imgui.End()
}
