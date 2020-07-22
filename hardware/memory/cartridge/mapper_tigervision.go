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

package cartridge

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// from bankswitch_sizes.txt:
//
// -3F: Tigervision was the only user of this intresting method.  This works
// in a similar fashion to the above method; however, there are only 4 2K
// segments instead of 4 1K ones, and the ROM image is broken up into 4 2K
// slices.  As before, the last 2K always points to the last 2K of the image.
// You select the desired bank by performing an STA $3F instruction.  The
// accumulator holds the desired bank number (0-3; only the lower two bits are
// used).  Any STA in the $00-$3F range will change banks.  This appears to
// interfere with the TIA addresses, which it does; however you just use $40 to
// $7F instead! :-)  $3F does not have a corresponding TIA register, so writing
// here has no effect other than switching banks.  Very clever; especially
// since you can implement this with only one chip! (a 74LS173)

type tigervision struct {
	mappingID   string
	description string

	// tigervision cartridges traditionally have 4 of banks of 2048 bytes. but
	// it can theoretically support anything up to 512 banks
	bankSize int
	banks    [][]uint8

	// tigervision cartridges divide memory into two 2k segments
	//  o the last segment always points to the last bank
	//  o the first segment can point to any of the other three
	//
	// the bank pointed to by the first segment is changed through the listen()
	// function (part of the implementation of the cartMapper interface).
	segment [2]int
}

// should work with any size cartridge that is a multiple of 2048
//  - tested with 8k (Miner2049 etc.) and 32k (Genesis_Egypt demo)
func newTigervision(data []byte) (cartMapper, error) {
	cart := &tigervision{
		mappingID:   "3F",
		description: "tigervision",
		bankSize:    2048,
	}

	if len(data)%cart.bankSize != 0 {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number bytes in the cartridge data", cart.mappingID))
	}

	numBanks := len(data) / cart.bankSize
	cart.banks = make([][]uint8, numBanks)

	for k := 0; k < numBanks; k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	cart.Initialise()

	return cart, nil
}

func (cart tigervision) String() string {
	return fmt.Sprintf("%s [%s] Banks: %d, %d", cart.mappingID, cart.description, cart.segment[0], cart.segment[1])
}

// ID implements the cartMapper interface
func (cart tigervision) ID() string {
	return cart.mappingID
}

// Initialise implements the cartMapper interface
func (cart *tigervision) Initialise() {
	cart.segment[0] = cart.NumBanks() - 2

	// the last segment always points to the last bank
	cart.segment[1] = cart.NumBanks() - 1
}

// Read implements the cartMapper interface
func (cart *tigervision) Read(addr uint16, _ bool) (uint8, error) {
	var data uint8
	if addr >= 0x0000 && addr <= 0x07ff {
		data = cart.banks[cart.segment[0]][addr&0x07ff]
	} else if addr >= 0x0800 && addr <= 0x0fff {
		data = cart.banks[cart.segment[1]][addr&0x07ff]
	}
	return data, nil
}

// Write implements the cartMapper interface
func (cart *tigervision) Write(addr uint16, data uint8, _ bool, poke bool) error {
	if poke {
		if addr >= 0x0000 && addr <= 0x07ff {
			cart.banks[cart.segment[0]][addr&0x07ff] = data
		} else if addr >= 0x0800 && addr <= 0x0fff {
			cart.banks[cart.segment[1]][addr&0x07ff] = data
		}
	}
	return errors.New(errors.MemoryBusError, addr)
}

// NumBanks implements the cartMapper interface
func (cart tigervision) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the cartMapper interface
func (cart *tigervision) GetBank(addr uint16) banks.Details {
	if addr >= 0x0000 && addr <= 0x07ff {
		return banks.Details{Number: cart.segment[0], IsRAM: false, Segment: 0}
	}
	return banks.Details{Number: cart.segment[1], IsRAM: false, Segment: 1}
}

// Patch implements the cartMapper interface
func (cart *tigervision) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return errors.New(errors.CartridgePatchOOB, offset)
	}

	bank := int(offset) / cart.bankSize
	offset = offset % cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the cartMapper interface
func (cart *tigervision) Listen(addr uint16, data uint8) {
	// tigervision is seemingly unique in that it bank switches when an address
	// outside of cartridge space is written to. for this to work, we need the
	// listen() function.
	//
	// update: tigervision is not unique. the 3e+ mapper also uses this
	// mechanism

	// although address 3F is used primarily, in actual fact writing anywhere
	// in TIA space is okay. from  the description from Kevin Horton's document
	// (quoted above) whenever an address in TIA space is written to, the lower
	// 3 bits of the value being written is used to set the segment.

	// !!TODO: lint check for data writes that specify a bank > NumBanks(). the
	// format allows this but it might be a mistake

	// bankswitch on hotspot access
	if addr < 0x40 {
		cart.segment[0] = int(data & uint8(cart.NumBanks()-1))
	}

	// this bank switching method causes a problem when the CPU wants to write
	// to TIA space for real and not cause a bankswitch. for this reason,
	// tigervision cartridges use mirror addresses to write to the TIA.
}

// Step implements the cartMapper interface
func (cart *tigervision) Step() {
}

// IterateBank implemnts the disassemble interface
func (cart tigervision) IterateBanks(prev *banks.Content) *banks.Content {
	b := prev.Number + 1
	if b >= 0 && b < len(cart.banks)-1 {
		// banks 0 to 6 can occupy any of the three segments
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart,
				memorymap.OriginCart + uint16(cart.bankSize),
				memorymap.OriginCart + uint16(cart.bankSize)*2,
			},
		}
	} else if b == len(cart.banks)-1 {
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				// cannot point to the first segment
				memorymap.OriginCart + uint16(cart.bankSize),
				memorymap.OriginCart + uint16(cart.bankSize)*2,
				memorymap.OriginCart + uint16(cart.bankSize)*3,
			},
		}
	}
	return nil
}
