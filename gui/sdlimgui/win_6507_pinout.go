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
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
)

const win6507PinoutID = "6507 Pinout"

type win6507Pinout struct {
	debuggerWin

	img *SdlImgui

	busInfoHeight float32

	// Vec4 colors for address and data bus. used as the basis for the
	// packed colors and for styling text
	addressBus    imgui.Vec4
	addressBusOff imgui.Vec4
	dataBus       imgui.Vec4
	dataBusOff    imgui.Vec4

	// packed colours are used for the drawlist primitives used to create the
	// pinout image
	body          imgui.PackedColor
	bodyOutline   imgui.PackedColor
	pinOn         imgui.PackedColor
	pinOff        imgui.PackedColor
	addressPinOn  imgui.PackedColor
	addressPinOff imgui.PackedColor
	dataPinOn     imgui.PackedColor
	dataPinOff    imgui.PackedColor
	rdyPinOn      imgui.PackedColor
	rdyPinOff     imgui.PackedColor
}

func newWin6507Pinout(img *SdlImgui) (window, error) {
	win := &win6507Pinout{
		img: img,
	}

	win.addressBus = imgui.Vec4{X: 0.3, Y: 0.8, Z: 0.8, W: 1.0}
	win.addressBusOff = imgui.Vec4{X: 0.3, Y: 0.8, Z: 0.8, W: 0.5}
	win.dataBus = imgui.Vec4{X: 0.8, Y: 0.8, Z: 0.3, W: 1.0}
	win.dataBusOff = imgui.Vec4{X: 0.8, Y: 0.8, Z: 0.3, W: 0.5}

	win.body = imgui.PackedColorFromVec4(imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.1, W: 1.0})
	win.bodyOutline = imgui.PackedColorFromVec4(imgui.Vec4{X: 1.0, Y: 1.0, Z: 1.0, W: 0.8})
	win.pinOn = imgui.PackedColorFromVec4(imgui.Vec4{X: 0.8, Y: 0.8, Z: 0.8, W: 1.0})
	win.pinOff = imgui.PackedColorFromVec4(imgui.Vec4{X: 0.8, Y: 0.8, Z: 0.8, W: 0.5})
	win.addressPinOn = imgui.PackedColorFromVec4(win.addressBus)
	win.addressPinOff = imgui.PackedColorFromVec4(win.addressBusOff)
	win.dataPinOn = imgui.PackedColorFromVec4(win.dataBus)
	win.dataPinOff = imgui.PackedColorFromVec4(win.dataBusOff)
	win.rdyPinOn = imgui.PackedColorFromVec4(win.img.cols.True)
	win.rdyPinOff = imgui.PackedColorFromVec4(win.img.cols.False)

	return win, nil
}

func (win *win6507Pinout) init() {
}

func (win *win6507Pinout) id() string {
	return win6507PinoutID
}

func (win *win6507Pinout) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 756, Y: 117}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 326, Y: 338}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{X: 326, Y: 338}, imgui.Vec2{X: 650, Y: 650})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *win6507Pinout) draw() {
	avail := imgui.ContentRegionAvail()
	avail.Y -= win.busInfoHeight
	p := imgui.CursorScreenPos()

	// size and positioning
	chipDim := imgui.Vec2{X: avail.X * 0.5, Y: avail.Y * 0.9}
	chipPos := imgui.Vec2{X: p.X + avail.X*0.5 - chipDim.X*0.5, Y: p.Y + avail.Y*0.5 - chipDim.Y*0.5}

	if imgui.BeginChildV("pinout", avail, false, imgui.WindowFlagsNone) {
		dl := imgui.WindowDrawList()
		imgui.PushFont(win.img.fonts.diagram)

		const lineThick = 2.0

		// main body
		dl.AddRectFilledV(chipPos, imgui.Vec2{X: chipPos.X + chipDim.X, Y: chipPos.Y + chipDim.Y},
			win.body, 0, imgui.DrawCornerFlagsAll)

		// pins
		pinSpacing := chipDim.Y / 14
		pinSize := pinSpacing / 2
		pinTextAdj := (pinSize - imgui.TextLineHeight()) / 2

		// address/data values (for convenience)
		addressBus := win.img.cache.VCS.Mem.AddressBus
		dataBus := win.img.cache.VCS.Mem.DataBus

		// left pins
		pinX := chipPos.X - pinSize
		for i := 0; i < 14; i++ {
			col := win.pinOff
			label := ""
			switch i {
			case 0:
				// RES
				if !win.img.cache.VCS.CPU.HasReset() {
					col = win.pinOn
				}
				label = "RES"
			case 1:
				// Vss
				col = win.pinOn
				label = "Vss"
			case 2:
				// RDY
				if win.img.cache.VCS.CPU.RdyFlg {
					col = win.rdyPinOn
				} else {
					col = win.rdyPinOff
				}
				label = "RDY"
			case 3:
				// Vcc
				col = win.pinOn
				label = "Vcc"
			default:
				// address pins
				m := uint16(0x01 << (i - 4))
				if addressBus&m == m {
					col = win.addressPinOn
				} else {
					col = win.addressPinOff
				}
				label = fmt.Sprintf("A%d", i-4)
			}

			pinY := chipPos.Y + pinSize*0.5 + (float32(i) * pinSpacing)
			pinPos := imgui.Vec2{X: pinX, Y: pinY}
			dl.AddRectFilledV(pinPos, imgui.Vec2{X: pinPos.X + pinSize, Y: pinPos.Y + pinSize},
				col, 0, imgui.DrawCornerFlagsNone)

			textPos := imgui.Vec2{X: chipPos.X + lineThick + chipDim.X*0.025, Y: pinPos.Y + pinTextAdj}
			dl.AddText(textPos, col, label)
		}

		pinX = chipPos.X + chipDim.X
		for i := 0; i < 14; i++ {
			col := win.pinOff
			label := ""
			switch i {
			case 0:
				switch win.img.cache.VCS.TIA.PClk {
				case phaseclock.RisingPhi2:
					col = win.pinOn
				case phaseclock.FallingPhi2:
					col = win.pinOn
				}
				label = "φ2"
			case 1:
				switch win.img.cache.VCS.TIA.PClk {
				case phaseclock.RisingPhi1:
					col = win.pinOn
				case phaseclock.FallingPhi1:
					col = win.pinOn
				}
				label = "φ1"
			case 2:
				// R/W
				if win.img.cache.VCS.Mem.LastCPUWrite {
					col = win.pinOn
				}
				label = "R/W"
			default:
				if i > 10 {
					// address pins
					m := uint16(0x01 << (23 - i))
					if addressBus&m == m {
						col = win.addressPinOn
					} else {
						col = win.addressPinOff
					}
					label = fmt.Sprintf("A%d", (23 - i))
				} else {
					// data pins
					m := uint16(0x01 << (i - 3))
					if uint16(dataBus)&m == m {
						col = win.dataPinOn
					} else {
						col = win.dataPinOff
					}
					label = fmt.Sprintf("D%d", i-3)
				}
			}

			pinY := chipPos.Y + pinSize*0.5 + (float32(i) * pinSpacing)
			pinPos := imgui.Vec2{X: pinX, Y: pinY}
			dl.AddRectFilledV(pinPos, imgui.Vec2{X: pinPos.X + pinSize, Y: pinPos.Y + pinSize},
				col, 0, imgui.DrawCornerFlagsNone)

			textPos := imgui.Vec2{X: chipPos.X + chipDim.X + lineThick*2 - imguiGetFrameDim(label).X, Y: pinPos.Y + pinTextAdj}
			dl.AddText(textPos, col, label)
		}

		// main chip body (outline)
		dl.AddRectV(chipPos, imgui.Vec2{X: chipPos.X + chipDim.X, Y: chipPos.Y + chipDim.Y},
			win.bodyOutline, 0, imgui.DrawCornerFlagsAll, lineThick)

		imgui.PopFont()
	}
	imgui.EndChild()

	// bus information
	win.busInfoHeight = imguiMeasureHeight(func() {
		if imgui.CollapsingHeaderV("Bus", imgui.TreeNodeFlagsDefaultOpen) {
			if imgui.BeginTableV("trackerHeader", 4, imgui.TableFlagsBordersInnerV|imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0) {
				// weight the column widths
				width := imgui.ContentRegionAvail().X
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, width*0.23, 0)
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, width*0.32, 1)
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, width*0.15, 2)
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, width*0.30, 3)

				imgui.TableNextRow()
				imgui.TableNextColumn()
				imguiColorLabelSimple("Address", win.addressBus)

				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.addressBus)
				imgui.Text(fmt.Sprintf("%013b", win.img.cache.VCS.Mem.AddressBus&0x1fff))
				imgui.PopStyleColor()

				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%#04x", win.img.cache.VCS.Mem.AddressBus&0x1fff))

				imgui.TableNextColumn()
				_, area := memorymap.MapAddress(win.img.cache.VCS.Mem.AddressBus, !win.img.cache.VCS.Mem.LastCPUWrite)
				imgui.Text(area.String())

				imgui.TableNextRow()
				imgui.TableNextColumn()
				imguiColorLabelSimple("Data", win.dataBus)

				imgui.TableNextColumn()
				if win.img.cache.VCS.Mem.DataBusDriven != 0xff {
					p := imgui.CursorScreenPos()
					s1 := strings.Builder{}
					s2 := strings.Builder{}
					for i := 7; i >= 0; i-- {
						if (win.img.cache.VCS.Mem.DataBusDriven>>i)&0x01 == 0x01 {
							s1.WriteString(fmt.Sprintf("%d", (win.img.cache.VCS.Mem.DataBus>>i)&0x01))
							s2.WriteRune(' ')
						} else {
							s1.WriteRune(' ')
							s2.WriteString(fmt.Sprintf("%d", (win.img.cache.VCS.Mem.DataBus>>i)&0x01))
						}
					}
					imgui.PushStyleColor(imgui.StyleColorText, win.dataBus)
					imgui.Text(s1.String())
					imgui.SetCursorScreenPos(p)
					imgui.PushStyleColor(imgui.StyleColorText, win.dataBusOff)
					imgui.Text(s2.String())
					imgui.PopStyleColorV(2)
				} else {
					imgui.PushStyleColor(imgui.StyleColorText, win.dataBus)
					imgui.Text(fmt.Sprintf("%08b", win.img.cache.VCS.Mem.DataBus))
					imgui.PopStyleColor()
				}

				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%#02x", win.img.cache.VCS.Mem.DataBus))

				imgui.TableNextColumn()
				if win.img.cache.VCS.Mem.LastCPUWrite {
					imgui.Text("Writing")
				} else {
					imgui.Text("Reading")
				}

				imgui.EndTable()
			}
		}
	})
}
