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

package ace

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

type aceMemory struct {
	model memorymodel.Map
	sram  []byte
	flash []byte

	resetSP uint32
	resetLR uint32
	resetPC uint32

	gpio []byte
}

const (
	aceHeaderMagic         = 0
	aceHeaderDriverName    = 9
	aceHeaderDriverVersion = 24
	aceHeaderROMSize       = 28
	aceHeaderROMChecksum   = 32
	aceHeaderEntryPoint    = 36
	startOfRom             = 40
)

const (
	GPIO_ADDR_IN  = 0x10
	GPIO_DATA_OUT = 0x14
	GPIO_MEMTOP   = 0xff
)

func newAceMemory(version string, data []byte) (*aceMemory, error) {
	mem := &aceMemory{}

	switch version {
	case "ACE-2600":
		return nil, curated.Errorf("ACE: unocart not yet supported")
	case "ACE-PC00":
		mem.model = memorymodel.NewMap(memorymodel.PlusCart)
	default:
		return nil, curated.Errorf("ACE: unrecognised version (%s)", version)
	}

	mem.resetSP = mem.model.SRAMOrigin | 0x00001fdc
	mem.resetLR = mem.model.FlashOrigin
	mem.resetPC = (uint32(data[aceHeaderEntryPoint])) |
		(uint32(data[aceHeaderEntryPoint+1]) << 8) |
		(uint32(data[aceHeaderEntryPoint+2]) << 16) |
		(uint32(data[aceHeaderEntryPoint+3]) << 24)
	mem.resetPC += mem.model.FlashOrigin

	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin)
	mem.flash = make([]byte, len(data))
	copy(mem.flash, data)

	mem.gpio = make([]byte, GPIO_MEMTOP)
	mem.gpio[GPIO_DATA_OUT] = 0xea

	return mem, nil
}

func (mem *aceMemory) Snapshot() *aceMemory {
	m := *mem
	return &m
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *aceMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr < GPIO_MEMTOP {
		return &mem.gpio, addr
	}
	if addr >= mem.model.FlashOrigin && addr <= mem.model.FlashOrigin+uint32(len(mem.flash)) {
		return &mem.flash, addr - mem.model.FlashOrigin
	}
	if addr >= mem.resetPC && addr <= mem.resetPC+uint32(len(mem.flash)) {
		return &mem.flash, addr - mem.resetPC + 0x1028
	}
	if addr >= mem.model.SRAMOrigin && addr <= mem.resetSP {
		return &mem.sram, addr - mem.model.SRAMOrigin
	}
	return nil, 0
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *aceMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}

// Ace implements the mapper.CartMapper interface.
type Ace struct {
	instance  *instance.Instance
	version   string
	pathToROM string
	arm       *arm.ARM
	mem       *aceMemory
}

// NewAce is the preferred method of initialisation for the Ace type.
func NewAce(instance *instance.Instance, pathToROM string, version string, data []byte) (mapper.CartMapper, error) {
	cart := &Ace{
		instance:  instance,
		version:   version,
		pathToROM: pathToROM,
	}

	var err error
	cart.mem, err = newAceMemory(version, data)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(arm.ARMv7_M, arm.MAMfull, cart.mem.model, cart.instance.Prefs.ARM, cart.mem, cart, cart.pathToROM)
	cart.arm.AddReadWatch(GPIO_ADDR_IN)
	cart.arm.Run()

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Ace) MappedBanks() string {
	return fmt.Sprintf("Bank: none")
}

// ID implements the mapper.CartMapper interface.
func (cart *Ace) ID() string {
	return fmt.Sprintf("ACE (%s)", cart.version)
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Ace) Snapshot() mapper.CartMapper {
	n := *cart
	n.mem = cart.mem.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Ace) Plumb() {
	cart.arm.Plumb(cart.mem, cart)
}

// Reset implements the mapper.CartMapper interface.
func (cart *Ace) Reset() {
}

// Read implements the mapper.CartMapper interface.
func (cart *Ace) Read(addr uint16, passive bool) (uint8, error) {
	if passive {
		cart.Listen(0x1000|addr, 0x00)
	}

	return cart.mem.gpio[GPIO_DATA_OUT], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *Ace) Write(_ uint16, _ uint8, _, _ bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Ace) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Ace) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *Ace) Patch(_ int, _ uint8) error {
	return curated.Errorf("ACE: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *Ace) Listen(addr uint16, _ uint8) {
	cart.mem.gpio[GPIO_ADDR_IN] = uint8(addr)
	cart.mem.gpio[GPIO_ADDR_IN+1] = uint8(addr >> 8)
	cart.arm.Continue()
	cart.arm.Continue()
	cart.arm.Continue()
	cart.arm.Continue()
}

// Step implements the mapper.CartMapper interface.
func (cart *Ace) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Ace) CopyBanks() []mapper.BankContent {
	return nil
}

// implements arm.CartridgeHook interface.
func (cart *Ace) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}
