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

package developer

import (
	"debug/elf"
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

type loclistContext interface {
	coproc() mapper.CartCoProc
	framebase() (uint64, error)
}

type location struct {
	address   uint64
	addressOk bool
	value     uint32
	valueOk   bool
}

type dwarfOperator func(*loclist) (location, error)

type loclist struct {
	ctx      loclistContext
	sequence []dwarfOperator
	stack    []location
}

func newLoclistJustContext(ctx loclistContext) *loclist {
	return &loclist{
		ctx: ctx,
	}
}

type commitLoclist func(start, end uint64)

func newLoclistFromSingleValue(ctx loclistContext, addr uint64) (*loclist, error) {
	loc := &loclist{
		ctx: ctx,
	}
	op := func(loc *loclist) (location, error) {
		return location{
			value:   uint32(addr),
			valueOk: true,
		}, nil
	}
	loc.sequence = append(loc.sequence, op)
	return loc, nil
}

func newLoclistFromSingleOperator(ctx loclistContext, expr []uint8) (*loclist, error) {
	loc := &loclist{
		ctx: ctx,
	}
	op, n := decodeDWARFoperation(expr, 0, true)
	if n == 0 {
		return nil, fmt.Errorf("unknown expression operator %02x", expr[0])
	}
	loc.sequence = append(loc.sequence, op)
	return loc, nil
}

func newLoclist(ctx loclistContext, debug_loc *elf.Section, ptr int64, commit commitLoclist) (*loclist, error) {
	loc := &loclist{
		ctx: ctx,
	}

	// buffer for reading the .debug_log section
	b := make([]byte, 16)

	// "Location lists, which are used to describe objects that have a limited lifetime or change
	// their location during their lifetime. Location lists are more completely described below."
	// page 26 of "DWARF4 Standard"
	//
	// "Location lists are used in place of location expressions whenever the object whose location is
	// being described can change location during its lifetime. Location lists are contained in a separate
	// object file section called .debug_loc . A location list is indicated by a location attribute whose
	// value is an offset from the beginning of the .debug_loc section to the first byte of the list for the
	// object in question"
	// page 30 of "DWARF4 Standard"
	//
	// "loclistptr: This is an offset into the .debug_loc section (DW_FORM_sec_offset). It consists
	// of an offset from the beginning of the .debug_loc section to the first byte of the data making up
	// the location list for the compilation unit. It is relocatable in a relocatable object file, and
	// relocated in an executable or shared object. In the 32-bit DWARF format, this offset is a 4-
	// byte unsigned value; in the 64-bit DWARF format, it is an 8-byte unsigned value (see
	// Section 7.4)"
	// page 148 of "DWARF4 Standard"

	// "The applicable base address of a location list entry is determined by the closest preceding base
	// address selection entry (see below) in the same location list. If there is no such selection entry,
	// then the applicable base address defaults to the base address of the compilation unit (see
	// Section 3.1.1)"
	//
	// "A base address selection entry affects only the list in which it
	// is contained" page 31 of "DWARF4 Standard"
	var baseAddress uint64

	// function to read an address from the debug_loc data
	readAddress := func() uint64 {
		debug_loc.ReadAt(b[:4], ptr)
		a := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24
		ptr += 4
		return a
	}

	startAddress := readAddress()
	endAddress := readAddress()

	// read single location description for this start/end address
	debug_loc.ReadAt(b, ptr)

	// "The end of any given location list is marked by an end of list entry, which consists of a 0 for the
	// beginning address offset and a 0 for the ending address offset. A location list containing only an
	// end of list entry describes an object that exists in the source code but not in the executable
	// program". page 31 of "DWARF4 Standard"
	for !(startAddress == 0x0 && endAddress == 0x0) {
		// "A base address selection entry consists of:
		// 1. The value of the largest representable address offset (for example, 0xffffffff when the size of
		// an address is 32 bits).
		// 2. An address, which defines the appropriate base address for use in interpreting the beginning
		// and ending address offsets of subsequent entries of the location list"
		// page 31 of "DWARF4 Standard"
		if startAddress == 0xffffffff {
			baseAddress = endAddress
		} else {
			// reduce end address by one. this is because the value we've read "marks the
			// first address past the end of the address range over which the location is
			// valid" (page 30 of "DWARF4 Standard")
			endAddress -= 1

			// length of expression
			debug_loc.ReadAt(b, ptr)
			length := int(b[0])
			length |= int(b[1]) << 8
			ptr += 2

			// loop through stack operations
			for length > 0 {
				// read single location description for this start/end address
				debug_loc.ReadAt(b, ptr)
				r, n := decodeDWARFoperation(b, 0, length <= 5)
				if n == 0 {
					return nil, fmt.Errorf("unknown expression operator %02x", b[0])
				}

				// add resolver to variable
				loc.addOperator(r)

				// reduce length value
				length -= n

				// advance debug_loc pointer by length value
				ptr += int64(n)
			}

			// "A location list entry (but not a base address selection or end of list entry) whose beginning
			// and ending addresses are equal has no effect because the size of the range covered by such
			// an entry is zero". page 31 of "DWARF4 Standard"
			//
			// "The ending address must be greater than or equal to the beginning address"
			// page 30 of "DWARF4 Standard"
			if startAddress < endAddress {
				if commit != nil {
					commit(startAddress+baseAddress, endAddress+baseAddress)
				}
			}
		}

		// read next address range
		startAddress = readAddress()
		endAddress = readAddress()
	}

	return loc, nil
}

func (loc *loclist) addOperator(r dwarfOperator) {
	loc.sequence = append(loc.sequence, r)
}

func (loc *loclist) resolve() (location, error) {
	if loc.ctx == nil {
		return location{}, fmt.Errorf("no context")
	}

	loc.stack = loc.stack[:0]
	for i := range loc.sequence {
		r, err := loc.sequence[i](loc)
		if err != nil {
			return location{}, err
		}
		if r.addressOk || r.valueOk {
			loc.stack = append(loc.stack, r)
		}
	}

	if len(loc.stack) == 0 {
		return location{}, fmt.Errorf("stack is empty")
	}

	return loc.stack[len(loc.stack)-1], nil
}

// lastResolved implements the resolver interface
func (loc *loclist) lastResolved() location {
	if len(loc.stack) == 0 {
		return location{}
	}
	return loc.stack[len(loc.stack)-1]
}

func (loc *loclist) pop() (location, bool) {
	l := len(loc.stack)
	if l == 0 {
		return location{}, false
	}
	r := loc.stack[l-1]
	loc.stack = loc.stack[:l-1]
	return r, true
}
