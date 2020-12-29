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

package arm7tdmi

type timer struct {
	active bool

	control uint32
	counter float32
}

// the ARM7TDMI in the Harmony runs at 70Mhz.
const armClock = float32(70)

func (t *timer) step(clock float32) {
	if !t.active {
		return
	}

	// the ARM timer ticks forward once every ARM cycle. the best we can do to
	// accommodate this is to tick the counter forward by the the appropriate
	// fraction every VCS cycle. Put another way: an NTSC spec VCS, for
	// example, will tick forward every 58-59 ARM cycles.
	t.counter += armClock / clock
}

func (t *timer) write(addr uint32, val uint32) bool {
	switch addr {
	case 0xe0008004:
		t.control = val
		t.active = t.control&0x01 == 0x01
	case 0xe0008008:
		t.counter = float32(val)
	default:
		return false
	}

	return true
}

func (t *timer) read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case 0xe0008004:
		val = t.control
	case 0xe0008008:
		val = uint32(t.counter)
	default:
		return 0, false
	}

	return val, true
}
