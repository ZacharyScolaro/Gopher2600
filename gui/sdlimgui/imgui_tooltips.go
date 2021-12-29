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

var tooltipColor = imgui.Vec4{0.08, 0.08, 0.08, 1.0}
var tooltipBorder = imgui.Vec4{0.03, 0.03, 0.03, 1.0}

// shows tooltip on hover of the previous imgui digest/group. useful for simple
// tooltips.
func imguiTooltipSimple(tooltip string) {
	// split string on newline and display with separate calls to imgui.Text()
	tooltip = strings.TrimSpace(tooltip)
	if tooltip != "" && imgui.IsItemHovered() {
		s := strings.Split(tooltip, "\n")
		imgui.PushStyleColor(imgui.StyleColorPopupBg, tooltipColor)
		imgui.PushStyleColor(imgui.StyleColorBorder, tooltipBorder)
		imgui.BeginTooltip()
		for _, t := range s {
			imgui.Text(t)
		}
		imgui.EndTooltip()
		imgui.PopStyleColorV(2)
	}
}

// shows tooltip that require more complex formatting than a single string.
//
// the hoverTest argument says that the tooltip should be displayed only
// when the last imgui widget/group is being hovered over with the mouse. if
// this argument is false then it implies that the decision to show the tooltip
// has already been made by the calling function.
func imguiTooltip(f func(), hoverTest bool) {
	if !hoverTest || imgui.IsItemHovered() {
		imgui.PushStyleColor(imgui.StyleColorPopupBg, tooltipColor)
		imgui.PushStyleColor(imgui.StyleColorBorder, tooltipBorder)
		imgui.BeginTooltip()
		f()
		imgui.EndTooltip()
		imgui.PopStyleColorV(2)
	}
}
