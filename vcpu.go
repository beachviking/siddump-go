package main

import (
	"fmt"

	"github.com/beevik/go6502/cpu"
)

func NewCpu() *cpu.CPU {
	mem := cpu.NewFlatMemory()
	CPU := cpu.NewCPU(cpu.NMOS, mem)
	return CPU
}

func InitCpu(cpu *cpu.CPU, newpc uint16, newa uint8, newx uint8, newy uint8) *cpu.CPU {
	cpu.SetPC(newpc)
	cpu.Reg.X = newx
	cpu.Reg.Y = newy
	cpu.Reg.A = newa
	return cpu
}

func RunCpu(cpu *cpu.CPU) uint8 {
	cpu.Step()

	// DumpCpuState(cpu)

	// Peek at the next opcode at the current PC
	opcode := cpu.Mem.LoadByte(cpu.Reg.PC)

	// Look up the instruction data for the opcode
	inst := cpu.InstSet.Lookup(opcode)

	switch {
	case (inst.Opcode == 0x00):
		return 0
	case (inst.Opcode == 0x40) && (cpu.Reg.SP == 0xFF):
		return 0
	case (inst.Opcode == 0x60) && (cpu.Reg.SP == 0xFF):
		return 0
	}

	return 1
}

func IncAtAddress(cpu *cpu.CPU, adr uint16) {
	cpu.Mem.StoreByte(adr, cpu.Mem.LoadByte(adr)+1)
}

func DumpCpuState(cpu *cpu.CPU) {
	fmt.Printf("PC: %04x OP: %02x A:%02x X:%02x Y:%02x\n", cpu.LastPC, cpu.Mem.LoadByte(cpu.LastPC), cpu.Reg.A, cpu.Reg.X, cpu.Reg.Y)
}
