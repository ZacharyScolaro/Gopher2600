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

package arm

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/logger"
)

func (arm *ARM) illegalAccess(event string, addr uint32) {
	arm.memoryError = fmt.Errorf("%s: unrecognised address %08x (PC: %08x)", event, addr, arm.state.instructionPC)

	if arm.dev == nil {
		return
	}

	detail := arm.dev.IllegalAccess(event, arm.state.instructionPC, addr)
	if detail != "" {
		arm.memoryErrorDev = fmt.Errorf("%s: %s", event, detail)
	}
}

// nullAccess is a special condition of illegalAccess()
func (arm *ARM) nullAccess(event string, addr uint32) {
	arm.memoryError = fmt.Errorf("%s: probable null pointer dereference of %08x (PC: %08x)", event, addr, arm.state.instructionPC)

	if arm.dev == nil {
		return
	}

	detail := arm.dev.NullAccess(event, arm.state.instructionPC, addr)
	if detail == "" {
		return
	}

	arm.memoryErrorDev = fmt.Errorf("%s: %s", event, detail)
}

// imperfect check of whether stack has collided with memtop
func (arm *ARM) stackCollision(stackPointerBeforeExecution uint32) (err error, detail error) {
	if arm.stackHasCollided || stackPointerBeforeExecution == arm.state.registers[rSP] {
		return
	}

	// check if stackMemory point and memtop are in the same memory block
	stackMemory, _ := arm.mem.MapAddress(arm.state.registers[rSP], true)
	variableMemory, _ := arm.mem.MapAddress(arm.variableMemtop, true)

	// check if the memory block and "variables" are in the same
	// memory block and that the stack pointer is below the top of
	// variable memory
	if stackMemory != variableMemory || arm.state.registers[rSP] > arm.variableMemtop {
		return
	}

	// set stackHasCollided flag. this means that memory accesses
	// will no longer be checked for legality
	arm.stackHasCollided = true

	err = fmt.Errorf("collision with program memory (%08x)", arm.state.registers[rSP])

	if arm.dev != nil {
		return
	}

	detailStr := arm.dev.StackCollision(arm.state.executingPC, arm.state.registers[rSP])
	if detailStr == "" {
		return
	}

	detail = fmt.Errorf("%s", detailStr)

	return err, detail
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 8bit", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Read 8bit", addr)
		}

		return uint8(arm.mmap.IllegalAccessValue)
	}

	return (*mem)[addr]
}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 8bit", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Write 8bit", addr)
		}

		return
	}

	(*mem)[addr] = val
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) read16bit(addr uint32, requiresAlignment bool) uint16 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 16bit", addr)
	}

	// check 16 bit alignment
	if requiresAlignment && addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit read (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Read 16bit", addr)
		}

		return uint16(arm.mmap.IllegalAccessValue)
	}

	// ensure we're not accessing past the end of memory
	if len(*mem) < 2 || addr >= uint32(len(*mem)-1) {
		arm.illegalAccess("Read 16bit", addr)
		return uint16(arm.mmap.IllegalAccessValue)
	}

	return arm.byteOrder.Uint16((*mem)[addr:])
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) write16bit(addr uint32, val uint16, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 16bit", addr)
	}

	// check 16 bit alignment
	if requiresAlignment && addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit write (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Write 16bit", addr)
		}

		return
	}

	// ensure we're not accessing past the end of memory
	if len(*mem) < 2 || addr >= uint32(len(*mem)-1) {
		arm.illegalAccess("Write 16bit", addr)
		return
	}

	arm.byteOrder.PutUint16((*mem)[addr:], val)
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) read32bit(addr uint32, requiresAlignment bool) uint32 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 32bit", addr)
	}

	// check 32 bit alignment
	if requiresAlignment && addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit read (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Read 32bit", addr)
		}

		return arm.mmap.IllegalAccessValue
	}

	// ensure we're not accessing past the end of memory
	if len(*mem) < 4 || addr >= uint32(len(*mem)-3) {
		arm.illegalAccess("Read 32bit", addr)
		return arm.mmap.IllegalAccessValue
	}

	return arm.byteOrder.Uint32((*mem)[addr:])
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) write32bit(addr uint32, val uint32, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 32bit", addr)
	}

	// check 32 bit alignment
	if requiresAlignment && addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit write (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		if !arm.stackHasCollided {
			arm.illegalAccess("Write 32bit", addr)
		}

		return
	}

	// ensure we're not accessing past the end of memory
	if len(*mem) < 4 || addr >= uint32(len(*mem)-3) {
		arm.illegalAccess("Write 32bit", addr)
		return
	}

	arm.byteOrder.PutUint32((*mem)[addr:], val)
}
