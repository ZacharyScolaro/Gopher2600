package video

import (
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
)

type collisions struct {
	cxm0p  uint8
	cxm1p  uint8
	cxp0fb uint8
	cxp1fb uint8
	cxm0fb uint8
	cxm1fb uint8
	cxblpf uint8
	cxppmm uint8
}

func (col *collisions) clear() {
	col.cxm0p = 0
	col.cxm1p = 0
	col.cxp0fb = 0
	col.cxp1fb = 0
	col.cxm0fb = 0
	col.cxm1fb = 0
	col.cxblpf = 0
	col.cxppmm = 0
}

func (col *collisions) SetMemory(mem memory.ChipBus) {
	mem.ChipWrite(symbols.CXM0P, col.cxm0p)
	mem.ChipWrite(symbols.CXM1P, col.cxm1p)
	mem.ChipWrite(symbols.CXP0FB, col.cxp0fb)
	mem.ChipWrite(symbols.CXP1FB, col.cxp1fb)
	mem.ChipWrite(symbols.CXM0FB, col.cxm0fb)
	mem.ChipWrite(symbols.CXM1FB, col.cxm1fb)
	mem.ChipWrite(symbols.CXBLPF, col.cxblpf)
	mem.ChipWrite(symbols.CXPPMM, col.cxppmm)
}
