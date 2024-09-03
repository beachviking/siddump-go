package main

import "github.com/beevik/go6502/cpu"

// Sid represents a SID chip.
type Sid struct {
	Channel [3]Voice
	Filt    Filter
	Register [27] byte
}

// Voice represents a voice in the SID chip.
type Voice struct {
	Freq  uint16
	Pulse uint16
	ADSR  uint16
	Wave  uint8
	Note  int
}

// Filter represents the filter in the SID chip.
type Filter struct {
	Type    uint8
	Control uint8
	Cutoff  uint16
}

func (v *Voice) Init() {
	v.Freq = 0
	v.Pulse = 0
	v.ADSR = 0
	v.Wave = 0
	v.Note = 0
}

func (v *Voice) CopyFrom(src *Voice) {
	v.Freq = src.Freq
	v.Pulse = src.Pulse
	v.ADSR = src.ADSR
	v.Wave = src.Wave
	v.Note = src.Note
}

func (f *Filter) Init() {
	f.Type = 0
	f.Control = 0
	f.Cutoff = 0
}

func (f *Filter) CopyFrom(src *Filter) {
	f.Type = src.Type
	f.Control = src.Control
	f.Cutoff = src.Cutoff
}

func NewSID() *Sid {
	sid := &Sid{}
	sid.Channel[0].Init()
	sid.Channel[1].Init()
	sid.Channel[2].Init()
	sid.Filt.Init()
	return sid
}

func (sid *Sid) CopyFrom(src *Sid) {
	sid.Channel[0].CopyFrom(&src.Channel[0])
	sid.Channel[1].CopyFrom(&src.Channel[1])
	sid.Channel[2].CopyFrom(&src.Channel[2])
	sid.Filt.CopyFrom(&src.Filt)

	copy(sid.Register[:], src.Register[:])
}

func (sid *Sid) CopyFromCpu(cpu *cpu.CPU) {
	// Get SID parameters from each channel and the filter
	for i := 0; i < 3; i++ {
		offset := uint16(7 * i)
		sid.Channel[i].Freq = uint16(cpu.Mem.LoadByte(0xD400+offset)) | (uint16(cpu.Mem.LoadByte(0xD401+offset)) << 8)
		sid.Channel[i].Pulse = uint16(cpu.Mem.LoadByte(0xD402+offset)) | (uint16(cpu.Mem.LoadByte(0xD403+offset))<<8)&0xFFF
		sid.Channel[i].Wave = uint8(cpu.Mem.LoadByte(0xD404 + offset))
		sid.Channel[i].ADSR = uint16(cpu.Mem.LoadByte(0xD406+offset)) | (uint16(cpu.Mem.LoadByte(0xD405+offset)) << 8)
	}

	sid.Filt.Cutoff = uint16(cpu.Mem.LoadByte(0xD415)<<5) | (uint16(cpu.Mem.LoadByte(0xD416)) << 8)
	sid.Filt.Control = uint8(cpu.Mem.LoadByte(0xD417))
	sid.Filt.Type = uint8(cpu.Mem.LoadByte(0xD418))

	for i := 0; i < 25; i++ {
		sid.Register[i] = cpu.Mem.LoadByte(uint16(0xD400+i))
	}

	if (cpu.Mem.LoadByte(0xdc05) == 0 && cpu.Mem.LoadByte(0xdc04) == 0) {
		// Most likely vbi driven, ie. 20000us
		sid.Register[25] = 0x4e; // dt HI
		sid.Register[26] = 0x20; // dt LO
	  } else {
		// CIA timer is used to control updates...
		sid.Register[25] = cpu.Mem.LoadByte(0xdc05) // dt HI
		sid.Register[26] = cpu.Mem.LoadByte(0xdc04) // dt LO
	  }
}
