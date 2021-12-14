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

package sdlimgui

import (
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
)

const winLogID = "Log"

type winLog struct {
	img  *SdlImgui
	open bool
}

func newWinLog(img *SdlImgui) (window, error) {
	win := &winLog{
		img: img,
	}

	return win, nil
}

func (win *winLog) init() {
}

func (win *winLog) id() string {
	return winLogID
}

func (win *winLog) isOpen() bool {
	return win.open
}

func (win *winLog) setOpen(open bool) {
	win.open = open
}

func (win *winLog) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{489, 352}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{570, 335}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.LogBackground)
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNone)
	imgui.PopStyleColor()

	var clipper imgui.ListClipper
	clipper.Begin(win.img.lz.Log.NumLines)
	for clipper.Step() {
		for i := clipper.DisplayStart; i < clipper.DisplayEnd && i < len(win.img.lz.Log.Entries); i++ {
			sp := strings.Split(win.img.lz.Log.Entries[i].String(), "\n")
			imgui.Text(sp[0])
			if len(sp) > 1 {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.LogMultilineEmphasis)
				for _, s := range sp[1:] {
					imgui.Text(s)
				}
				imgui.PopStyleColor()
			}
		}
	}

	// scroll to end if log has been dirtied (ie. a new entry)
	if win.img.lz.Log.Dirty {
		imgui.SetScrollHereY(1.0)
		win.img.lz.Log.Dirty = false
	}

	imgui.End()
}
