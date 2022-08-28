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
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winTIA) drawMissile(missile int) {
	imgui.BeginChildV(fmt.Sprintf("##missile%d", missile), imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.footerHeight}, false, 0)
	defer imgui.EndChild()

	lz := win.img.lz.Missile0
	ms := win.img.lz.Missile0.Ms
	ps := win.img.lz.Player0.Ps
	if missile != 0 {
		lz = win.img.lz.Missile1
		ms = win.img.lz.Missile1.Ms
		ps = win.img.lz.Player1.Ps
	}

	imgui.Spacing()

	imgui.BeginGroup()
	imguiLabel("Colour")
	col := lz.Color
	if win.img.imguiSwatch(col, 0.75) {
		win.popupPalette.request(&col, func() {
			win.img.dbg.PushFunction(func() { ms.Color = col })

			// update player color too
			win.img.dbg.PushFunction(func() { ps.Color = col })
		})
	}

	imguiLabel("Reset-to-Player")
	r2p := lz.ResetToPlayer
	if imgui.Checkbox("##resettoplayer", &r2p) {
		win.img.dbg.PushFunction(func() { ms.ResetToPlayer = r2p })
	}

	imgui.SameLine()
	imguiLabel("Enabled")
	enb := lz.Enabled
	if imgui.Checkbox("##enabled", &enb) {
		win.img.dbg.PushFunction(func() { ms.Enabled = enb })
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// hmove value and slider
	imgui.BeginGroup()
	imguiLabel("HMOVE")
	imgui.SameLine()
	hmove := fmt.Sprintf("%01x", lz.Hmove)
	if imguiHexInput("##hmove", 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { ms.Hmove = uint8(v) })
		}
	}

	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(lz.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d", imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() { ms.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// nusiz
	imgui.BeginGroup()
	imgui.PushItemWidth(win.missileCopiesComboDim.X)
	if imgui.BeginComboV("##missilecopies", video.MissileCopies[lz.Copies], imgui.ComboFlagsNoArrowButton) {
		for k := range video.MissileCopies {
			if imgui.Selectable(video.MissileCopies[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushFunction(func() {
					ms.Copies = v
					win.img.vcs.TIA.Video.UpdateNUSIZ(missile, true)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imgui.PushItemWidth(win.missileSizeComboDim.X)
	if imgui.BeginComboV("##missilesize", video.MissileSizes[lz.Size], imgui.ComboFlagsNoArrowButton) {
		for k := range video.MissileSizes {
			if imgui.Selectable(video.MissileSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushFunction(func() {
					ms.Size = v
					win.img.vcs.TIA.Video.UpdateNUSIZ(missile, true)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiLabel("NUSIZ")
	nusiz := fmt.Sprintf("%02x", lz.Nusiz)
	if imguiHexInput("##nusiz", 2, &nusiz) {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() {
				ms.SetNUSIZ(uint8(v))

				// update player NUSIZ too
				ps.SetNUSIZ(uint8(v))
			})
		}
	}

	s := strings.Builder{}
	if lz.EncActive {
		s.WriteString("drawing ")
		if lz.EncSecondHalf {
			s.WriteString("2nd half of ")
		}
		switch lz.EncCpy {
		case 0:
			s.WriteString("1st copy")
		case 1:
			s.WriteString("2nd copy")
		case 2:
			s.WriteString("3rd copy")
		}
		s.WriteString(fmt.Sprintf(" [%d]", lz.EncTicks))
	}
	imgui.SameLine()
	imgui.Text(s.String())
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.BeginGroup()
	imgui.Text(fmt.Sprintf("Last reset at clock %03d. First copy draws at clock %03d", lz.ResetPixel, lz.HmovedPixel))
	if lz.MoreHmove {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
	imgui.EndGroup()
}
