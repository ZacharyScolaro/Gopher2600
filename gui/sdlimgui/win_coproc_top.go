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
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcTopID = "Coprocessor Top"
const winCoProcTopMenu = "Top"

type winCoProcTop struct {
	img           *SdlImgui
	open          bool
	showSrc       bool
	optionsHeight float32
}

func newWinCoProcTop(img *SdlImgui) (window, error) {
	win := &winCoProcTop{
		img:     img,
		showSrc: true,
	}
	return win, nil
}

func (win *winCoProcTop) init() {
}

func (win *winCoProcTop) id() string {
	return winCoProcTopID
}

func (win *winCoProcTop) isOpen() bool {
	return win.open
}

func (win *winCoProcTop) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcTop) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcTopID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	// safely iterate over top execution information
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		imgui.BeginChildV("##coprocTopMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)
		imgui.BeginTabBar("##coprocSourceTabBar")

		if imgui.BeginTabItemV("Previous Frame", nil, imgui.TabItemFlagsNone) {
			win.drawExecutionTop(src, false)
			imgui.EndTabItem()
		}

		if imgui.BeginTabItemV("Lifetime", nil, imgui.TabItemFlagsNone) {
			win.drawExecutionTop(src, true)
			imgui.EndTabItem()
		}

		imgui.EndTabBar()
		imgui.EndChild()

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()
			imgui.Checkbox("Show Source in Tooltip", &win.showSrc)
		})
	})
}

func (win *winCoProcTop) drawExecutionTop(src *developer.Source, byLifetimeCycles bool) {
	src.Resort(byLifetimeCycles)

	const top = 25

	imgui.Spacing()
	imgui.BeginTableV("##coprocTopTable", 5, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 0)
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsNone, width*0.35, 1)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNone, width*0.1, 2)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNone, width*0.35, 3)
	imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNone, width*0.1, 4)

	if src == nil {
		imgui.Text("No source files available")
		return
	}

	imgui.TableHeadersRow()

	for i := 0; i < top; i++ {
		imgui.TableNextRow()
		ln := src.ExecutedLines.Lines[i]

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// source on tooltip
		if win.showSrc {
			imguiTooltip(func() {
				imgui.Text(ln.File.Filename)
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
				imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
				imgui.PopStyleColor()
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.Text(strings.TrimSpace(ln.Content))
			}, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSource(ln)
		}

		imgui.TableNextColumn()
		imgui.Text(ln.File.Filename)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if ln.Function == "" {
			imgui.Text(developer.UnknownFunction)
		} else {
			imgui.Text(fmt.Sprintf("%s()", ln.Function))
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if byLifetimeCycles {
			if src.TotalCycles > 0 {
				imgui.Text(fmt.Sprintf("%0.2f%%", ln.LifetimeCycles/src.TotalCycles*100.0))
			}
		} else {
			if src.FrameCycles > 0 {
				imgui.Text(fmt.Sprintf("%0.2f%%", ln.FrameCycles/src.FrameCycles*100.0))
			}
		}
		imgui.PopStyleColor()
	}

	imgui.EndTable()
}
