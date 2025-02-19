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

package plusrom

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// sentinal errors indicating a specific problem with the attempt to load the
// child cartridge into the PlusROM.
var NotAPlusROM = errors.New("not a plus rom")
var CannotAdoptROM = errors.New("cannot adopt ROM")

// PlusROM wraps another mapper.CartMapper inside a network aware format.
type PlusROM struct {
	env *environment.Environment

	net   *network
	state *state

	// rewind boundary is indicated on every network activity
	rewindBoundary bool
}

// rewindable state for the 3e cartridge.
type state struct {
	child mapper.CartMapper
}

// Snapshot implements the mapper.CartMapper interface.
func (s *state) Snapshot() *state {
	n := *s
	n.child = s.child.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (s *state) Plumb(env *environment.Environment) {
	s.child.Plumb(env)
}

func NewPlusROM(env *environment.Environment, child mapper.CartMapper, romfile io.ReadSeeker) (mapper.CartMapper, error) {
	cart := &PlusROM{env: env}
	cart.state = &state{}
	cart.state.child = child

	cart.net = newNetwork(cart.env)

	// host/path information are found at the address pointed to by an address
	// stored near the end of the ROM file. this is roughly equivalent to the
	// NMI vector of the last bank stored in the ROM file

	var b [2]byte

	_, err := romfile.Seek(-6, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("plusrom: %w: %w", CannotAdoptROM, err)
	}
	n, err := romfile.Read(b[:])
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("plusrom: %w: %w", CannotAdoptROM, err)
		}
	}
	if n != 2 {
		return nil, fmt.Errorf("plusrom: %w: invalid NMI vector", CannotAdoptROM)
	}

	// the address of the host/path information
	hostPathAddr := (uint16(b[1]) << 8) | uint16(b[0])

	// the retrieved address is defined to be relative to memory origin 0x1000
	// but is actually an offset into the ROM file. correcting the origin
	// address by subtracting 0x1000
	hostPathAddr -= 0x1000

	logger.Logf(env, "plusrom", "host address at ROM offset 0x%04x", hostPathAddr)

	_, err = romfile.Seek(int64(hostPathAddr), io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("plusrom: %w: %w", CannotAdoptROM, err)
	}

	// read path string from the first bank using the indirect address retrieved above
	path := strings.Builder{}
	for path.Len() < maxPathLength {
		_, err := romfile.Read(b[:1])
		if err != nil {
			return nil, fmt.Errorf("plusrom: %w: %w", CannotAdoptROM, err)
		}
		if b[0] == 0x00 {
			break // for loop
		}
		path.WriteRune(rune(b[0]))
	}

	// read host string. this string continues on from the path string. the
	// address pointer will be in the correct place.
	host := strings.Builder{}
	for host.Len() <= maxHostLength {
		_, err := romfile.Read(b[:1])
		if err != nil {
			return nil, fmt.Errorf("plusrom: %w: %w", CannotAdoptROM, err)
		}
		if b[0] == 0x00 {
			break // for loop
		}
		host.WriteRune(rune(b[0]))
	}

	// fail if host or path is not valid
	hostValid, pathValid := cart.SetAddrInfo(host.String(), path.String())
	if !hostValid || !pathValid {
		return nil, fmt.Errorf("%w: invalid host/path", NotAPlusROM)
	}

	// log success
	logger.Logf(env, "plusrom", "will connect to %s", cart.net.ai.String())

	if cart.env.Prefs.PlusROM.NewInstallation {
		err := cart.env.Notifications.Notify(notifications.NotifyPlusROMNewInstall)
		if err != nil {
			return nil, fmt.Errorf("plusrom %w:", err)
		}
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) MappedBanks() string {
	return cart.state.child.MappedBanks()
}

// ID implements the mapper.CartMapper interface.
func (cart *PlusROM) ID() string {
	// not altering the underlying cartmapper's ID
	return cart.state.child.ID()
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *PlusROM) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *PlusROM) Plumb(env *environment.Environment) {
	cart.env = env
	cart.state.Plumb(env)
}

// ID implements the mapper.CartContainer interface.
func (cart *PlusROM) ContainerID() string {
	return "PlusROM"
}

// Reset implements the mapper.CartMapper interface.
func (cart *PlusROM) Reset() {
	cart.state.child.Reset()
}

// READ implements the mapper.CartMapper interface.
func (cart *PlusROM) Access(addr uint16, peek bool) (data uint8, mask uint8, err error) {
	switch addr {
	case 0x0ff2:
		// 1FF2 contains the next byte of the response from the host, every
		// read will increment the receive buffer pointer (receive buffer is
		// max 256 bytes also!)
		return cart.net.recv(), mapper.CartDrivenPins, nil

	case 0x0ff3:
		// 1FF3 contains the number of (unread) bytes left in the receive buffer
		// (these bytes can be from multiple responses)
		return uint8(cart.net.recvRemaining()), mapper.CartDrivenPins, nil
	}

	return cart.state.child.Access(addr, peek)
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *PlusROM) AccessVolatile(addr uint16, data uint8, poke bool) error {
	switch addr {
	case 0x0ff0:
		// 1FF0 is for writing a byte to the send buffer (max 256 bytes)
		cart.net.buffer(data)
		return nil

	case 0x0ff1:
		// 1FF1 is for writing a byte to the send buffer and submit the buffer
		// to the back end API
		cart.rewindBoundary = true
		cart.net.buffer(data)
		cart.net.commit()
		err := cart.env.Notifications.Notify(notifications.NotifyPlusROMNetwork)
		if err != nil {
			return fmt.Errorf("plusrom %w:", err)
		}
		return nil
	}

	return cart.state.child.AccessVolatile(addr, data, poke)
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) NumBanks() int {
	return cart.state.child.NumBanks()
}

// GetBank implements the mapper.CartMapper interface.
func (cart *PlusROM) GetBank(addr uint16) mapper.BankInfo {
	return cart.state.child.GetBank(addr)
}

// SetBank implements the mapper.CartMapper interface.
func (cart *PlusROM) SetBank(bank string) error {
	if cart, ok := cart.state.child.(mapper.SelectableBank); ok {
		return cart.SetBank(bank)
	}
	return fmt.Errorf("plusrom: %s does not support setting of bank", cart.state.child.ID())
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *PlusROM) AccessPassive(addr uint16, data uint8) error {
	return cart.state.child.AccessPassive(addr, data)
}

// Step implements the mapper.CartMapper interface.
func (cart *PlusROM) Step(clock float32) {
	cart.net.transmitWait()
	cart.state.child.Step(clock)
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) CopyBanks() []mapper.BankContent {
	return cart.state.child.CopyBanks()
}

// GetGetRegisters implements the mapper.CartRegistersBus interface.
func (cart *PlusROM) GetRegisters() mapper.CartRegisters {
	if cart, ok := cart.state.child.(mapper.CartRegistersBus); ok {
		return cart.GetRegisters()
	}
	return nil
}

// PutRegister implements the mapper.CartRegistersBus interface.
func (cart *PlusROM) PutRegister(register string, data string) {
	if cart, ok := cart.state.child.(mapper.CartRegistersBus); ok {
		cart.PutRegister(register, data)
	}
}

// GetRAM implements the mapper.CartRAMbus interface.
func (cart *PlusROM) GetRAM() []mapper.CartRAM {
	if cart, ok := cart.state.child.(mapper.CartRAMbus); ok {
		return cart.GetRAM()
	}
	return nil
}

// PutRAM implements the mapper.CartRAMbus interface.
func (cart *PlusROM) PutRAM(bank int, idx int, data uint8) {
	if cart, ok := cart.state.child.(mapper.CartRAMbus); ok {
		cart.PutRAM(bank, idx, data)
	}
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *PlusROM) GetStatic() mapper.CartStatic {
	if cart, ok := cart.state.child.(mapper.CartStaticBus); ok {
		return cart.GetStatic()
	}
	return nil
}

// PutStatic implements the mapper.CartStaticBus interface.
func (cart *PlusROM) PutStatic(segment string, idx int, data uint8) bool {
	if cart, ok := cart.state.child.(mapper.CartStaticBus); ok {
		return cart.PutStatic(segment, idx, data)
	}
	return true
}

// Rewind implements the mapper.CartTapeBus interface.
func (cart *PlusROM) Rewind() {
	if cart, ok := cart.state.child.(mapper.CartTapeBus); ok {
		cart.Rewind()
	}
}

// GetTapeState implements the mapper.CartTapeBus interface.
func (cart *PlusROM) GetTapeState() (bool, mapper.CartTapeState) {
	if cart, ok := cart.state.child.(mapper.CartTapeBus); ok {
		return cart.GetTapeState()
	}
	return false, mapper.CartTapeState{}
}

// Patch implements the mapper.CartPatchable interface.
func (cart *PlusROM) Patch(offset int, data uint8) error {
	if cart, ok := cart.state.child.(mapper.CartPatchable); ok {
		return cart.Patch(offset, data)
	}
	return nil
}

// RewindBoundary implements the mapper.CartRewindBoundary interface.
func (cart *PlusROM) RewindBoundary() bool {
	if cart.rewindBoundary {
		cart.rewindBoundary = false
		return true
	}
	return false
}
