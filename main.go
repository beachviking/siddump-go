package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
)

const MAX_INSTR uint16 = 0xFFFF

func main() {
	opt := SidOutputSettings{}
	var frame int = 0

	// Parse arguments
	opt.ParseArgs()

	// Open the SID file
	if len(flag.Args()) == 0 {
		fmt.Println("Usage: go run main.go [options] <sidfile>")
		os.Exit(1)
	}

	if opt.Usage == 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	sidName := flag.Arg(0)

	// Try to open SID file
	file, err := os.Open(sidName)
	check(err)
	defer file.Close()

	// Read interesting parts of the SID header
	file.Seek(6, 0)
	dataOffset := readWord(file)
	loadAddress := readWord(file)
	initAddress := readWord(file)
	playAddress := readWord(file)

	file.Seek(int64(dataOffset), 0)
	if loadAddress == 0 {
		loadAddress = uint16(readByte(file)) | uint16(readByte(file))<<8
	}

	fmt.Printf("dataOffset:  0x%X\n", dataOffset)
	fmt.Printf("loadAddress: 0x%X\n", loadAddress)
	fmt.Printf("initAddress: 0x%X\n", initAddress)
	fmt.Printf("playAddress: 0x%X\n", playAddress)

	// Load the C64 data
	filePos, fileErr := file.Seek(0, io.SeekCurrent)
	check(fileErr)
	loadPos := filePos
	filePos, fileErr = file.Seek(0, io.SeekEnd)
	check(fileErr)
	loadEnd := uint16(filePos)
	loadSize := uint16(loadEnd) - uint16(loadPos)
	file.Seek(int64(loadPos), 0)

	if loadSize+loadAddress >= 0x10000-1 {
		panic("Error: SID data continues past end of C64 memory.")
	}

	memPos := uint16(loadAddress)
	cpu := NewCpu()

	for {
		var b byte

		fileErr = binary.Read(file, binary.LittleEndian, &b)

		if fileErr == io.EOF {
			break
		}
		cpu.Mem.StoreByte(memPos, b)
		// fmt.Printf("Adr %04X val %02X\n", memPos, b)
		memPos++
	}

	// Print info and run initroutine
	fmt.Printf("Load address: $%04X Init address: $%04X Play address: $%04X\n", loadAddress, initAddress, playAddress)
	fmt.Printf("Calling initroutine with subtune %d\n", opt.Subtune)
	cpu.Mem.StoreByte(0x01, 0x37)
	InitCpu(cpu, initAddress, uint8(opt.Subtune), 0, 0)
	instr := 0

	for RunCpu(cpu) == 1 {
		IncAtAddress(cpu, 0xD012)
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

	if playAddress == 0 {
		fmt.Println("Warning: SID has play address 0, reading from interrupt vector instead")
		if cpu.Mem.LoadByte(0x01)&0x07 == 0x5 {
			playAddress = uint16(cpu.Mem.LoadByte(0xFFFE)) | (uint16(cpu.Mem.LoadByte(0xFFFF)) << 8)
		} else {
			playAddress = uint16(cpu.Mem.LoadByte(0x314)) | (uint16(cpu.Mem.LoadByte(0x315)) << 8)
		}
		fmt.Printf("New play address is $%04X\n", playAddress)
	}

	currentSid := NewSID()

	// Create requested output struct type
	screenSidReg := &ScreenOutputSidRegisters{Options: &opt, SidState: currentSid}
	screenNotes := &ScreenOutputWithNotes{Options: &opt, SidState: currentSid}

	output := &ActiveDecoder{}

	switch (opt.DecoderOutput) {
	case 1:
		output.SetOutput(screenSidReg)
	default:
		output.SetOutput(screenNotes)
	}

	fmt.Printf("Calling playroutine for %d frames, starting from frame %d\n", opt.Seconds*50, opt.Firstframe)

	output.PreProcess()

	for frame < opt.Firstframe+opt.Seconds*50 {
		// Run the playroutine
		instr = 0
		InitCpu(cpu, playAddress, 0, 0, 0)

		for RunCpu(cpu) == 1 {
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

	fmt.Println("Simulation done!")

}
