package main

// may be usefull to look at binary dumps in the terminal:
// od -h sidtune.dmp | less

import (
	"flag"
	"fmt"
	"os"
)

const MAX_INSTR uint16 = 0xFFFF

func main() {
	// opt := SidOutputSettings{}
	opt := NewSidOutputSettings()
	header := NewPSID()
	var frame int = 0

	// Parse arguments
	opt.ParseArgs()

	if len(flag.Args()) == 0 {
		fmt.Println("Usage: go run main.go [options] <sidfile>")
		os.Exit(1)
	}

	if opt.Usage == 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// get file name of sid tune
	sidName := flag.Arg(0)

	// Try to open SID file
	file, err := os.Open(sidName)
	check(err)
	defer file.Close()

	// Load PSID header
	err = header.LoadPSIDHeader(file)
	check(err)

	header.PrintPSIDVitals()

	// Load PSID data into cpu memory
	cpu := NewCpu()
	err = header.LoadPSIDData(cpu, file)
	check(err)

	// Print info and run initroutine
	fmt.Printf("Load address: $%04X Init address: $%04X Play address: $%04X\n", header.LoadAddress, header.InitAddress, header.PlayAddress)
	fmt.Printf("Calling initroutine with subtune %d\n", opt.Subtune)
	cpu.Mem.StoreByte(0x01, 0x37)
	Init(cpu, header.InitAddress, uint8(opt.Subtune), 0, 0)
	instr := 0

	for Run(cpu) == 1 {
		IncrementValueAtAddress(cpu, 0xD012)
		if (cpu.Mem.LoadByte(0xD012) == 0) || (((cpu.Mem.LoadByte(0xD011) & 0x80) != 0) && (cpu.Mem.LoadByte(0xd012) >= 0x38)) {
			tmp := cpu.Mem.LoadByte(0xD011)
			tmp ^= 0x80
			cpu.Mem.StoreByte(0xD011, tmp)
			cpu.Mem.StoreByte(0xD012, 0x0)
		}
		instr += 1

		if instr > int(MAX_INSTR) {
			fmt.Println("Warning: CPU executed a high number of instructions in init, breaking")
			break
		}
	}

	if header.PlayAddress == 0 {
		fmt.Println("Warning: SID has play address 0, reading from interrupt vector instead")
		if cpu.Mem.LoadByte(0x01)&0x07 == 0x5 {
			header.PlayAddress = uint16(cpu.Mem.LoadByte(0xFFFE)) | (uint16(cpu.Mem.LoadByte(0xFFFF)) << 8)
		} else {
			header.PlayAddress = uint16(cpu.Mem.LoadByte(0x314)) | (uint16(cpu.Mem.LoadByte(0x315)) << 8)
		}
		fmt.Printf("New play address is $%04X\n", header.PlayAddress)
	}

	currentSid := NewSID()

	// Create requested output struct type
	screenSidReg := &ScreenOutputSidRegisters{Options: opt, SidState: currentSid}
	screenNotes := &ScreenOutputWithNotes{Options: opt, SidState: currentSid}
	fileSidDtDump := &BinFileRegistersAndDtDumps{Options: opt, SidState: currentSid, fileName: "sidtune.dmp"}

	output := &ActiveDecoder{}

	switch opt.DecoderOutput {
	case 1:
		output.SetOutput(screenSidReg)
	case 4:
		output.SetOutput(fileSidDtDump)
	default:
		output.SetOutput(screenNotes)
	}

	fmt.Printf("Calling playroutine for %d frames, starting from frame %d\n", opt.Seconds*50, opt.Firstframe)

	output.PreProcess()

	for frame < opt.Firstframe+opt.Seconds*50 {
		// Run the playroutine
		instr = 0
		Init(cpu, header.PlayAddress, 0, 0, 0)

		for Run(cpu) == 1 {
			instr += 1

			if instr > int(MAX_INSTR) {
				fmt.Println("Warning: CPU executed a high number of instructions in init, breaking")
				break
			}

			// Test for jump into Kernal interrupt handler exit
			if ((cpu.Mem.LoadByte(0x01) & 0x07) != 0x5) && (cpu.Reg.PC == 0xEA31 || cpu.Reg.PC == 0xEA81) {
				break
			}
		}

		// // Update Sid with latest values from memory
		currentSid.CopyFromCpu(cpu)

		// Frame display
		if frame >= opt.Firstframe {
			output.ProcessFrame(frame, cpu.Cycles)
		}

		// Advance to next frame
		frame++
	}

	output.PostProcess()
}
