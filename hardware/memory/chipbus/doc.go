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

// Package chipbus defines the operations, addresses and symbols that are
// required by the TIA and RIOT chips when accessing memory.
//
// Another way to think of this is: the operations, etc. that are required to
// interface with the VCS memory from the perspective of the TIA and RIOT
// chips.
//
// It also defines the operations required by the TIA and RIOT chips in order
// to respond to changes made by the CPU.
package chipbus
