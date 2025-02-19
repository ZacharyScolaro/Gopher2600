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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey/i2c"
)

const winSaveKeyI2CID = "SaveKey I2C"
const winSaveKeyI2CMenu = "I2C"

type winSaveKeyI2C struct {
	debuggerWin

	img *SdlImgui

	// savekey instance
	savekey *savekey.SaveKey
}

func newWinSaveKeyI2C(img *SdlImgui) (window, error) {
	win := &winSaveKeyI2C{
		img: img,
	}

	return win, nil
}

func (win *winSaveKeyI2C) init() {
}

func (win *winSaveKeyI2C) id() string {
	return winSaveKeyI2CID
}

func (win *winSaveKeyI2C) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not draw if savekey is not active
	win.savekey = win.img.cache.VCS.GetSaveKey()
	if win.savekey == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 633, Y: 358}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSaveKeyI2C) draw() {
	win.drawStatus()

	imguiSeparator()

	win.drawAddress()
	imgui.SameLine()
	win.drawBits()
	imgui.SameLine()
	win.drawACK()

	imgui.Spacing()
	imgui.Spacing()

	win.drawOscilloscope()
}

func (win *winSaveKeyI2C) drawOscilloscope() {
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.SaveKeyOscBG)

	w := imgui.WindowWidth()
	w -= (imgui.CurrentStyle().FramePadding().X * 2) + (imgui.CurrentStyle().ItemInnerSpacing().X * 2)

	pos := imgui.CursorPos()
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.SaveKeyOscSCL)
	imgui.PlotLinesV("", win.savekey.SCL.Activity, 0, "", i2c.TraceLo, i2c.TraceHi,
		imgui.Vec2{X: w, Y: imgui.FrameHeight() * 2})

	// reset cursor pos with a slight offset
	pos.Y += 2.0
	imgui.SetCursorPos(pos)

	// transparent background color for second plotlines widget.
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.SaveKeyOscBG)

	// plot lines
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.SaveKeyOscSDA)
	imgui.PlotLinesV("", win.savekey.SDA.Activity, 0, "", i2c.TraceLo, i2c.TraceHi,
		imgui.Vec2{X: w, Y: imgui.FrameHeight() * 2})

	imgui.PopStyleColorV(4)

	// key to oscilloscope
	imgui.Spacing()
	imguiColorLabelSimple("SCL", win.img.cols.SaveKeyOscSCL)
	imgui.SameLine()
	imguiColorLabelSimple("SDA", win.img.cols.SaveKeyOscSDA)
}

func (win *winSaveKeyI2C) drawStatus() {
	imgui.AlignTextToFramePadding()
	switch win.savekey.State {
	case savekey.SaveKeyStopped:
		imgui.Text("Stopped")
	case savekey.SaveKeyStarting:
		imgui.Text("Starting")
	case savekey.SaveKeyAddressHi:
		fallthrough
	case savekey.SaveKeyAddressLo:
		imgui.Text("Getting address")
	case savekey.SaveKeyData:
		switch win.savekey.Dir {
		case savekey.Reading:
			imgui.Text("Reading")
		case savekey.Writing:
			imgui.Text("Writing")
		}
		imgui.SameLine()
		imgui.Text("Data")
	}
}

func (win *winSaveKeyI2C) drawACK() {
	v := win.savekey.Ack
	imgui.AlignTextToFramePadding()
	imgui.Text("ACK")
	imgui.SameLine()
	if imgui.Checkbox("##ACK", &v) {
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.Ack = v
			}
		})
	}
}

func (win *winSaveKeyI2C) drawBits() {
	bits := win.savekey.Bits
	bitCt := win.savekey.BitsCt

	var label string
	switch win.savekey.Dir {
	case savekey.Reading:
		label = "Reading"
	case savekey.Writing:
		label = "Writing"
	}

	s := fmt.Sprintf("%02x", bits)
	imguiLabel(label)
	if imguiHexInput(fmt.Sprintf("##%s", label), 2, &s) {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			panic(err)
		}
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.Bits = uint8(v)
			}
		})
	}

	imgui.SameLine()

	seq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, true)
	for i := 0; i < 8; i++ {
		if (bits<<i)&0x80 != 0x80 {
			seq.nextItemDepressed = true
		}
		if seq.rectFill(win.img.cols.saveKeyBit) {
			v := bits ^ (0x80 >> i)
			win.img.dbg.PushFunction(func() {
				if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
					sk.Bits = v
				}
			})
		}
		seq.sameLine()
	}
	seq.end()

	dl := imgui.WindowDrawList()
	dl.AddCircleFilled(imgui.Vec2{X: seq.offsetX(bitCt), Y: imgui.CursorScreenPos().Y},
		imgui.FontSize()*0.20, win.img.cols.saveKeyBitPointer)
}

func (win *winSaveKeyI2C) drawAddress() {
	addr := win.savekey.EEPROM.Address

	label := "Address"
	s := fmt.Sprintf("%04x", addr)
	imguiLabel(label)
	if imguiHexInput(fmt.Sprintf("##%s", label), 4, &s) {
		v, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			panic(err)
		}
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.EEPROM.Address = uint16(v)
			}
		})
	}
}
