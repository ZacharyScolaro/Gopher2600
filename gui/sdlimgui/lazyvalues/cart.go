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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

// LazyCart lazily accesses cartridge information from the emulator
type LazyCart struct {
	val *Lazy

	atomicID       atomic.Value // string
	atomicSummary  atomic.Value // string
	atomicFilename atomic.Value // string
	atomicNumBanks atomic.Value // int
	atomicCurrBank atomic.Value // int

	atomicStaticBus atomic.Value // mapper.CartStaticBus
	atomicStatic    atomic.Value // mapper.CartStatic

	atomicRegistersBus atomic.Value // mapper.CartRegistersBus
	atomicRegisters    atomic.Value // mapper.CartRegisters

	atomicRAMbus atomic.Value // mapper.CartRAMbus
	atomicRAM    atomic.Value // []mapper.CartRAM

	atomicTapeBus   atomic.Value // mapper.CartTapeBus
	atomicTapeState atomic.Value // mapper.CartTapeState

	atomicPlusROM         atomic.Value // plusrom.PlusROM
	atomicPlusROMAddrInfo atomic.Value // plusrom.AddrInfo
	atomicPlusROMNick     atomic.Value // string (from prefs.String.Get())
	atomicPlusROMID       atomic.Value // string (from prefs.String.Get())
	atomicPlusROMRecvBuff atomic.Value // []uint8
	atomicPlusROMSendBuff atomic.Value // []uint8

	ID       string
	Summary  string
	Filename string
	NumBanks int
	CurrBank mapper.BankInfo

	HasStaticBus bool
	StaticBus    mapper.CartStaticBus
	Static       []mapper.CartStatic

	HasRegistersBus bool
	RegistersBus    mapper.CartRegistersBus
	Registers       mapper.CartRegisters

	HasRAMbus bool
	RAMbus    mapper.CartRAMbus
	RAM       []mapper.CartRAM

	HasTapeBus bool
	TapeBus    mapper.CartTapeBus
	TapeState  mapper.CartTapeState

	IsPlusROM       bool
	PlusROMAddrInfo plusrom.AddrInfo
	PlusROMNick     string
	PlusROMID       string
	PlusROMRecvBuff []uint8
	PlusROMSendBuff []uint8
}

func newLazyCart(val *Lazy) *LazyCart {
	return &LazyCart{val: val}
}

func (lz *LazyCart) update() {
	// make a copy of CPU.PCaddr because we will be reading it in a different
	// goroutine (in the PushRawEvent() below) to the one in which it is
	// written (it is written to in the current thread in the LazyCPU.update()
	// function)
	PCaddr := lz.val.CPU.PCaddr

	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicID.Store(lz.val.Dbg.VCS.Mem.Cart.ID())
		lz.atomicFilename.Store(lz.val.Dbg.VCS.Mem.Cart.Filename)
		lz.atomicSummary.Store(lz.val.Dbg.VCS.Mem.Cart.MappingSummary())
		lz.atomicNumBanks.Store(lz.val.Dbg.VCS.Mem.Cart.NumBanks())
		lz.atomicCurrBank.Store(lz.val.Dbg.VCS.Mem.Cart.GetBank(PCaddr))

		sb := lz.val.Dbg.VCS.Mem.Cart.GetStaticBus()
		if sb != nil {
			lz.atomicStaticBus.Store(sb)

			// make sure CartStaticBus implementation is meaningful
			a := sb.GetStatic()
			if a != nil {
				lz.atomicStatic.Store(a)
			}
		}

		rb := lz.val.Dbg.VCS.Mem.Cart.GetRegistersBus()
		if rb != nil {
			lz.atomicRegistersBus.Store(rb)

			// make sure CartRegistersBus implementation is meaningful
			a := rb.GetRegisters()
			if a != nil {
				lz.atomicRegisters.Store(a)
			}
		}

		r := lz.val.Dbg.VCS.Mem.Cart.GetRAMbus()
		if r != nil {
			lz.atomicRAMbus.Store(r)

			// make sure CartRAMBus implementation is meaningful
			a := r.GetRAM()
			if a != nil {
				lz.atomicRAM.Store(a)
			}
		}

		t := lz.val.Dbg.VCS.Mem.Cart.GetTapeBus()
		if t != nil {
			// make sure CartTapeBus implementation is meaningful
			if ok, s := t.GetTapeState(); ok {
				lz.atomicTapeBus.Store(t)
				lz.atomicTapeState.Store(s)
			}
		}

		c := lz.val.Dbg.VCS.Mem.Cart.GetContainer()
		if c != nil {
			if pr, ok := c.(*plusrom.PlusROM); ok {
				lz.atomicPlusROM.Store(pr)
				lz.atomicPlusROMAddrInfo.Store(pr.CopyAddrInfo())
				lz.atomicPlusROMNick.Store(pr.Prefs.Nick.Get())
				lz.atomicPlusROMID.Store(pr.Prefs.ID.Get())
				lz.atomicPlusROMRecvBuff.Store(pr.CopyRecvBuffer())
				lz.atomicPlusROMSendBuff.Store(pr.CopySendBuffer())
			} else {
				lz.atomicPlusROM.Store(nil)
			}
		}
	})

	lz.ID, _ = lz.atomicID.Load().(string)
	lz.Summary, _ = lz.atomicSummary.Load().(string)
	lz.Filename, _ = lz.atomicFilename.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)
	lz.CurrBank, _ = lz.atomicCurrBank.Load().(mapper.BankInfo)

	lz.StaticBus, lz.HasStaticBus = lz.atomicStaticBus.Load().(mapper.CartStaticBus)
	if lz.HasStaticBus {
		lz.Static, _ = lz.atomicStatic.Load().([]mapper.CartStatic)

		// a cartridge can implement a static bus but not actually have a
		// static area. this additional test checks for that
		//
		// * required for PlusROM cartridges
		if lz.Static == nil {
			lz.HasStaticBus = false
		}
	}

	lz.RegistersBus, lz.HasRegistersBus = lz.atomicRegistersBus.Load().(mapper.CartRegistersBus)
	if lz.HasRegistersBus {
		lz.Registers, _ = lz.atomicRegisters.Load().(mapper.CartRegisters)

		// a cartridge can implement a registers bus but not actually have any
		// registers. this additional test checks for that
		//
		// * required for:
		//		- PlusROM cartridges
		if lz.Registers == nil {
			lz.HasRegistersBus = false
		}
	}

	lz.RAMbus, lz.HasRAMbus = lz.atomicRAMbus.Load().(mapper.CartRAMbus)
	if lz.HasRAMbus {
		lz.RAM, _ = lz.atomicRAM.Load().([]mapper.CartRAM)

		// a cartridge can implement a ram bus but not actually have any ram.
		// this additional test checks for that
		//
		// * required for:
		//		- atari cartridges without a superchip
		//		- PlusROM cartridges
		if lz.RAM == nil {
			lz.HasRAMbus = false
		}
	}

	lz.TapeBus, lz.HasTapeBus = lz.atomicTapeBus.Load().(mapper.CartTapeBus)
	if lz.HasTapeBus {
		lz.TapeState, _ = lz.atomicTapeState.Load().(mapper.CartTapeState)
	}

	_, lz.IsPlusROM = lz.atomicPlusROM.Load().(*plusrom.PlusROM)
	if lz.IsPlusROM {
		lz.PlusROMAddrInfo, _ = lz.atomicPlusROMAddrInfo.Load().(plusrom.AddrInfo)
		lz.PlusROMNick, _ = lz.atomicPlusROMNick.Load().(string)
		lz.PlusROMID, _ = lz.atomicPlusROMID.Load().(string)
		lz.PlusROMRecvBuff, _ = lz.atomicPlusROMRecvBuff.Load().([]uint8)
		lz.PlusROMSendBuff, _ = lz.atomicPlusROMSendBuff.Load().([]uint8)
	}
}
