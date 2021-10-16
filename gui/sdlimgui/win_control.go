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

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"

	"github.com/inkyblackness/imgui-go/v4"
)

const winControlID = "Control"

type winControl struct {
	img  *SdlImgui
	open bool
}

func newWinControl(img *SdlImgui) (window, error) {
	win := &winControl{
		img: img,
	}
	return win, nil
}

func (win *winControl) init() {
}

func (win *winControl) id() string {
	return winControlID
}

func (win *winControl) isOpen() bool {
	return win.open
}

func (win *winControl) setOpen(open bool) {
	win.open = open
}

func (win *winControl) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{665, 272}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// running
	win.drawRunButton()

	// stepping
	imgui.Spacing()
	win.drawStep()

	// fps
	imgui.Separator()
	imgui.Spacing()
	win.drawFPS()

	// mouse capture button
	imguiSeparator()
	imgui.Spacing()
	win.drawMouseCapture()

	imgui.End()
}

func (win *winControl) drawRunButton() {
	runDim := imgui.Vec2{X: imguiRemainingWinWidth(), Y: imgui.FrameHeight()}
	if win.img.emulation.State() == emulation.Running {
		if imguiBooleanButton(win.img.cols, false, fmt.Sprintf("%c Halt", fonts.Halt), runDim) {
			win.img.term.pushCommand("HALT")
		}
	} else {
		if imguiBooleanButton(win.img.cols, true, fmt.Sprintf("%c Run", fonts.Run), runDim) {
			win.img.term.pushCommand("RUN")
		}
	}
}

func (win *winControl) drawStep() {
	fillWidth := imgui.Vec2{X: -1, Y: imgui.FrameHeight()}

	if imgui.BeginTable("step", 2) {
		imgui.TableSetupColumnV("step", imgui.TableColumnFlagsWidthFixed, 75, 1)
		imgui.TableNextRow()

		// step button
		imgui.TableNextColumn()

		if imgui.Button(fmt.Sprintf("%c ##Step", fonts.Back)) {
			win.img.term.pushCommand("STEP BACK")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Step", fillWidth) {
			win.img.term.pushCommand("STEP")
		}

		imgui.TableNextColumn()

		if imguiToggleButton("##quantumToggle", win.img.lz.Debugger.Quantum == debugger.QuantumVideo, win.img.cols.TitleBgActive) {
			if win.img.lz.Debugger.Quantum == debugger.QuantumVideo {
				win.img.term.pushCommand("QUANTUM INSTRUCTION")
			} else {
				win.img.term.pushCommand("QUANTUM VIDEO")
			}
		}

		imgui.SameLine()
		imgui.AlignTextToFramePadding()
		if win.img.lz.Debugger.Quantum == debugger.QuantumVideo {
			imgui.Text("Video Clock")
		} else {
			imgui.Text("CPU Instruction")
		}

		imgui.EndTable()
	}

	if imgui.BeginTable("stepframescanline", 2) {
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 2)
		imgui.TableNextRow()
		imgui.TableNextColumn()

		if imgui.Button(fmt.Sprintf("%c ##Frame", fonts.Back)) {
			win.img.term.pushCommand("STEP BACK FRAME")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Frame", fillWidth) {
			win.img.term.pushCommand("STEP FRAME")
		}

		imgui.TableNextColumn()

		if imgui.Button(fmt.Sprintf("%c ##Scanline", fonts.Back)) {
			win.img.term.pushCommand("STEP BACK SCANLINE")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Scanline", fillWidth) {
			win.img.term.pushCommand("STEP SCANLINE")
		}

		imgui.EndTable()
	}

	if imgui.ButtonV("Step Over", fillWidth) {
		win.img.term.pushCommand("STEP OVER")
	}
}

func (win *winControl) drawFPS() {
	imgui.Text("Performance")
	imgui.Spacing()

	w := imguiRemainingWinWidth()
	imgui.PushItemWidth(w)
	defer imgui.PopItemWidth()

	// fps slider
	fps := win.img.lz.TV.ReqFPS
	if imgui.SliderFloatV("##fps", &fps, 1, 100, "%.0f fps", imgui.SliderFlagsNone) {
		win.img.dbg.PushRawEvent(func() { win.img.vcs.TV.SetFPS(fps) })
	}

	// reset to specification rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.dbg.PushRawEvent(func() { win.img.vcs.TV.SetFPS(-1) })
	}

	imgui.Spacing()
	if win.img.emulation.State() == emulation.Running {
		if win.img.lz.TV.ActualFPS <= win.img.lz.TV.ReqFPS*0.95 {
			imgui.Text("running below requested FPS")
		} else if win.img.lz.TV.ActualFPS > win.img.lz.TV.ReqFPS*0.95 {
			imgui.Text("running above requested FPS")
		}
	} else if win.img.lz.TV.ReqFPS < win.img.lz.TV.FrameInfo.Spec.RefreshRate {
		imgui.Text(fmt.Sprintf("below %s rate of %.0f fps", win.img.lz.TV.FrameInfo.Spec.ID, win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	} else if win.img.lz.TV.ReqFPS > win.img.lz.TV.FrameInfo.Spec.RefreshRate {
		imgui.Text(fmt.Sprintf("above %s rate of %.0f fps", win.img.lz.TV.FrameInfo.Spec.ID, win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	} else {
		imgui.Text(fmt.Sprintf("selected %s rate of %.0f fps", win.img.lz.TV.FrameInfo.Spec.ID, win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	}
}

func (win *winControl) drawMouseCapture() {
	imgui.AlignTextToFramePadding()
	imgui.Text(string(fonts.Mouse))
	imgui.SameLine()
	if win.img.wm.dbgScr.isCaptured {
		imgui.AlignTextToFramePadding()
		imgui.Text("RMB to release mouse")
	} else if imgui.Button("Capture mouse") {
		win.img.setCapture(true)
	}
}
