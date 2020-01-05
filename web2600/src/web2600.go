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

// +build js
// +build wasm

package main

import (
	"gopher2600/cartridgeloader"
	"gopher2600/hardware"
	"gopher2600/hardware/riot/input"
	"syscall/js"
)

func main() {
	worker := js.Global().Get("self")
	scr := NewCanvas(worker)

	// create new vcs
	vcs, err := hardware.NewVCS(scr)
	if err != nil {
		panic(err)
	}

	// load cartridge
	cartload := cartridgeloader.Loader{
		Filename: "http://localhost:2600/example.bin",
	}

	err = vcs.AttachCartridge(cartload)
	if err != nil {
		panic(err)
	}

	// add message handler - implements controllers
	messageHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var err error

		data := args[0].Get("data")
		switch data.Get("cmd").String() {
		case "keyDown":
			key := data.Get("key").Int()
			switch key {
			case 37: // left
				err = vcs.Player0.Handle(input.Left)
			case 39: // right
				err = vcs.Player0.Handle(input.Right)
			case 38: // up
				err = vcs.Player0.Handle(input.Up)
			case 40: // down
				err = vcs.Player0.Handle(input.Down)
			case 32: // space
				err = vcs.Player0.Handle(input.Fire)
			}
		case "keyUp":
			key := data.Get("key").Int()
			switch key {
			case 37: // left
				err = vcs.Player0.Handle(input.NoLeft)
			case 39: // right
				err = vcs.Player0.Handle(input.NoRight)
			case 38: // up
				err = vcs.Player0.Handle(input.NoUp)
			case 40: // down
				err = vcs.Player0.Handle(input.NoDown)
			case 32: // space
				err = vcs.Player0.Handle(input.NoFire)
			}
		default:
			js.Global().Get("self").Call("log", args[0].String())
		}

		if err != nil {
			panic(err)
		}

		return nil
	})
	defer func() {
		worker.Call("removeEventListener", "message", messageHandler, false)
		messageHandler.Release()
	}()
	worker.Call("addEventListener", "message", messageHandler, false)

	// run emulation
	vcs.Run(func() (bool, error) {
		return true, nil
	})
}
