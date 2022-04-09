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

// IllegalAccessEntry is a single entry in the illegal access log.
type IllegalAccessEntry struct {
	Event        string
	PC           uint32
	AccessAddr   uint32
	IsNullAccess bool
	SrcLine      *SourceLine
	Count        int
}

// IllegalAccess records memory accesses by the coprocesser that are "illegal".
type IllegalAccess struct {
	// entries are keyed by PC/AccessAddr expressed as a 16 character string
	entries map[string]*IllegalAccessEntry

	// all the accesses in order of the first time they appear. the Count field
	// in the IllegalAccessEntry can be used to see if that entry was seen more
	// than once *after* the first appearance
	Log []*IllegalAccessEntry
}
