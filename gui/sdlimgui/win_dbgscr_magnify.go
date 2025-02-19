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
	"image"

	"github.com/inkyblackness/imgui-go/v4"
)

const magnifyMax = 3
const magnifyMin = 20
const magnifyDef = 10

type dbgScrMagnifyTooltip struct {
	// whether to show magnification in the tooltip
	showInTooltip bool

	// textures
	texture texture

	// area of magnification for tooltip
	clip image.Rectangle

	// the amount of zoom in the tooltip magnification
	zoom int
}

func (mag *dbgScrMagnifyTooltip) draw(mouse dbgScrMouse) {
	if !mag.showInTooltip {
		return
	}

	_, delta := imgui.CurrentIO().MouseWheel()
	if delta < 0 && mag.zoom < magnifyMin {
		mag.zoom++
	} else if delta > 0 && mag.zoom > magnifyMax {
		mag.zoom--
	}

	mag.clip = image.Rect(mouse.scaled.x-mag.zoom,
		mouse.scaled.y-mag.zoom*pixelWidth,
		mouse.scaled.x+mag.zoom,
		mouse.scaled.y+mag.zoom*pixelWidth)

	// the magnify texture should be opaque
	imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 1.0)
	imgui.Image(imgui.TextureID(mag.texture.getID()), imgui.Vec2{X: 200, Y: 200})
	imgui.PopStyleVar()
}

type dbgScrMagnifyWindow struct {
	// whether magnification window is open
	open bool

	// textures
	texture texture

	// area of magnification for window this is used to clip the larger screen texture
	clip image.Rectangle

	// centre point of magnification area for window
	centerPoint dbgScrMousePos

	// the amount of zoom in the magnify window
	zoom int

	// dragging information
	isDragging    bool
	lastDragPoint dbgScrMousePos
}

func (mag *dbgScrMagnifyWindow) setClipCenter(centre dbgScrMouse) {
	mag.centerPoint = centre.scaled
	mag.setClip()
}

func (mag *dbgScrMagnifyWindow) setClip() {
	mag.clip.Min.X = mag.centerPoint.x - mag.zoom
	mag.clip.Min.Y = mag.centerPoint.y - mag.zoom*pixelWidth
	mag.clip.Max.X = mag.centerPoint.x + mag.zoom
	mag.clip.Max.Y = mag.centerPoint.y + mag.zoom*pixelWidth
}

func (mag *dbgScrMagnifyWindow) adjustZoom(delta float32) {
	if delta < 0 && mag.zoom < magnifyMin {
		mag.zoom++
	} else if delta > 0 && mag.zoom > magnifyMax {
		mag.zoom--
	}
	mag.setClip()
}

func (mag *dbgScrMagnifyWindow) adjustClip(drag dbgScrMousePos) {
	mag.centerPoint.x += drag.x
	mag.centerPoint.y += drag.y
	mag.setClip()
}

func (mag *dbgScrMagnifyWindow) draw(cols *imguiColors) {
	if !mag.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 8, Y: 28}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 200, Y: 200}, imgui.ConditionFirstUseEver)

	if imgui.BeginV("Magnification", &mag.open, imgui.WindowFlagsNoScrollbar) {
		// the size of single 2600 "pixel" as it is seen in the magnification
		// we use this to help with mouse dragging
		var pixelSize float32

		sz := imgui.ContentRegionAvail()
		if sz.X >= sz.Y {
			pixelSize = sz.Y / float32(mag.zoom*2)
			imgui.SetCursorPos(imgui.CursorPos().Plus(imgui.Vec2{X: (sz.X - sz.Y) / 2.0, Y: 0}))
			sz = imgui.Vec2{X: sz.Y, Y: sz.Y}
		} else {
			pixelSize = sz.X / float32(mag.zoom*2)
			imgui.SetCursorPos(imgui.CursorPos().Plus(imgui.Vec2{X: 0, Y: (sz.Y - sz.X) / 2.0}))
			sz = imgui.Vec2{X: sz.X, Y: sz.X}
		}

		imgui.PushStyleColor(imgui.StyleColorButton, cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, cols.Transparent)
		imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{X: 0.0, Y: 0.0})
		imgui.ImageButton(imgui.TextureID(mag.texture.getID()), sz)

		if imgui.IsItemHovered() || mag.isDragging {
			// adjust zoom with mouse wheel
			_, delta := imgui.CurrentIO().MouseWheel()
			if delta != 0 {
				mag.adjustZoom(delta)
			}

			// drag magnified area with mouse drag - left button or middle button
			if imgui.IsMouseDown(0) || imgui.IsMouseDown(2) {
				pos := imgui.MousePos()
				scaledPos := dbgScrMousePos{
					x: int(pos.X / pixelSize),
					y: int(pos.Y / pixelSize * pixelWidth),
				}

				if mag.isDragging {
					drag := dbgScrMousePos{
						x: mag.lastDragPoint.x - scaledPos.x,
						y: mag.lastDragPoint.y - scaledPos.y,
					}
					mag.adjustClip(drag)
				} else {
					mag.isDragging = true
				}

				mag.lastDragPoint = scaledPos
			} else {
				mag.isDragging = false
			}
		} else {
			mag.isDragging = false
		}

		imgui.PopStyleVar()
		imgui.PopStyleColorV(3)
	}

	imgui.End()
}
