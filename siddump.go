package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// type SidHeader struct {
//     Magic  [4]byte
// 	Version [2]byte
// 	dataOffset [2]byte
// 	loadAddress [2]byte
// 	initAddress [2]byte
// 	playAddress [2]byte
// 	Songs [2]byte
// 	StartSong [2]byte
// 	Speed [4]byte
// 	Name [32]byte
// 	Author [32]byte
// 	Released [32]byte
// }

// type SidData struct {
//     Length uint32
//     Data   []byte
// }

// type SidTune struct {
// 	SidHeader
// 	SidData
// }

// var (
// 	Memory [0x10000]byte
// )

const MAX_INSTR uint16 = 0xFFFF

func readbyte(f *os.File) byte {
	var res byte
	binary.Read(f, binary.LittleEndian, &res)
	return res
}

func readword(f *os.File) uint16 {
	var res [2]byte
	binary.Read(f, binary.LittleEndian, &res)

	word := uint16(res[0])<<8 | uint16(res[1])
	return word
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	var opt SidOutputOptions
	var frames int = 0
	var counter int = 0
	var rows int = 0

	// Scan arguments
	flag.IntVar(&opt.Subtune, "a", 0, "Accumulator value on init (subtune number) default = 0")
	flag.IntVar(&opt.Basefreq, "c", 0, "Frequency recalibration. Give note frequency in hex")
	flag.IntVar(&opt.Basenote, "d", 0xb0, "Select calibration note (abs.notation 80-DF). Default middle-C (B0)")
	flag.IntVar(&opt.Firstframe, "f", 0, "First frame to display, default 0")
	flag.IntVar(&opt.Lowres, "l", 1, "Low-resolution mode (only display 1 row per note)")
	flag.IntVar(&opt.Spacing, "n", 0, "Note spacing, default 0 (none)")
	flag.IntVar(&opt.Oldnotefactor, "o", 1, "'Oldnote-sticky' factor. Default 1, increase for better vibrato display")
	flag.IntVar(&opt.Pattspacing, "p", 0, "Pattern spacing, default 0 (none)")
	flag.IntVar(&opt.Timeseconds, "s", 0, "Display time in minutes:seconds:frame format")
	flag.IntVar(&opt.Seconds, "t", 60, "Playback time in seconds, default 60")
	flag.IntVar(&opt.Usage, "h", 0, "Display usage information")
	flag.IntVar(&opt.Profiling, "z", 0, "Include CPU cycles+rastertime (PAL)+rastertime, badline corrected")
	flag.Parse()

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
	file, err := os.Open(sidName)
	check(err)
	defer file.Close()

	// Read interesting parts of the SID header
	file.Seek(6, 0)
	dataOffset := readword(file)
	loadAddress := readword(file)
	initAddress := readword(file)
	playAddress := readword(file)

	file.Seek(int64(dataOffset), 0)
	if loadAddress == 0 {
		loadAddress = uint16(readbyte(file)) | uint16(readbyte(file))<<8
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

	for (RunCpu(cpu) == 1) {
		IncAtAddress(cpu, 0xD012)
		// if (!mem[0xd012] || ((mem[0xd011] & 0x80) && mem[0xd012] >= 0x38))
		if ((cpu.Mem.LoadByte(0xD012) == 0) || (((cpu.Mem.LoadByte(0xD011) & 0x80) != 0) && (cpu.Mem.LoadByte(0xd012) >= 0x38))) {
			tmp := cpu.Mem.LoadByte(0xD011)
			tmp ^= 0x80
			cpu.Mem.StoreByte(0xD011, tmp)
			cpu.Mem.StoreByte(0xD012, 0x0)
		}
		instr += 1
		
		if (instr > int(MAX_INSTR)) {
			fmt.Println("Warning: CPU executed a high number of instructions in init, breaking")
			break
		}
	}

	if (playAddress == 0) {
		fmt.Println("Warning: SID has play address 0, reading from interrupt vector instead")
		if (cpu.Mem.LoadByte(0x01) & 0x07 == 0x5) {
			playAddress = uint16(cpu.Mem.LoadByte(0xFFFE)) | (uint16(cpu.Mem.LoadByte(0xFFFF)) << 8)
		} else {
			playAddress = uint16(cpu.Mem.LoadByte(0x314)) | (uint16(cpu.Mem.LoadByte(0x315)) << 8)
		}
		fmt.Printf("New play address is $%04X\n", playAddress)
	}

	curr_sid := NewSID()
	prev_sid := NewSID()	
	prev2_sid := NewSID()

	fmt.Printf("Calling playroutine for %d frames, starting from frame %d\n", opt.Seconds*50, opt.Firstframe)
	fmt.Printf("Middle C frequency is $%04X\n\n", uint16(freqtbllo[48]) | (uint16(freqtblhi[48]) << 8));
	fmt.Printf("| Frame | Freq Note/Abs WF ADSR Pul | Freq Note/Abs WF ADSR Pul | Freq Note/Abs WF ADSR Pul | FCut RC Typ V |")

	if (opt.Profiling != 0) { 
		// CPU cycles, Raster lines, Raster lines with badlines on every 8th line, first line included
		fmt.Printf(" Cycl RL RB |");
	}
	fmt.Printf("\n");
	fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+");
	if (opt.Profiling != 0) {
	  fmt.Printf("------------+");
	}
	fmt.Printf("\n");
  
	for (frames < opt.Firstframe + opt.Seconds * 50) {
		// Run the playroutine
		instr = 0
		InitCpu(cpu, playAddress, 0, 0, 0)

		for (RunCpu(cpu) == 1) {
			instr += 1
		
			if (instr > int(MAX_INSTR)) {
				fmt.Println("Warning: CPU executed a high number of instructions in init, breaking")
				break
			}

			// Test for jump into Kernal interrupt handler exit
			if (((cpu.Mem.LoadByte(0x01) & 0x07) != 0x5) && (cpu.Reg.PC == 0xEA31 || cpu.Reg.PC == 0xEA81)) {
				break
			}			
		}

		// Get SID parameters from each channel and the filter
		for i := 0; i < 3; i++ {
			curr_sid.Channel[i].Freq = uint16(cpu.Mem.LoadByte(0xD400 + uint16(7*i))) | (uint16(cpu.Mem.LoadByte(0xD401 + uint16(7*i))) << 8)
			curr_sid.Channel[i].Pulse = uint16(cpu.Mem.LoadByte(0xD402 + uint16(7*i))) | (uint16(cpu.Mem.LoadByte(0xD403 + uint16(7*i))) << 8) & 0xFFF
			curr_sid.Channel[i].Wave = uint8(cpu.Mem.LoadByte(0xD404 + uint16(7*i)))
			curr_sid.Channel[i].ADSR = uint16(cpu.Mem.LoadByte(0xD406 + uint16(7*i))) | (uint16(cpu.Mem.LoadByte(0xD405 + uint16(7*i))) << 8)
		}

		curr_sid.Filt.Cutoff = uint16(cpu.Mem.LoadByte(0xD415) << 5) | (uint16(cpu.Mem.LoadByte(0xD416)) << 8)
		curr_sid.Filt.Control = uint8(cpu.Mem.LoadByte(0xD417)) 
		curr_sid.Filt.Type = uint8(cpu.Mem.LoadByte(0xD418))
		
		// Frame display
		if (frames >= opt.Firstframe) {
			var sb strings.Builder
			time := frames - opt.Firstframe

			if (opt.Timeseconds == 0) {
				sb.WriteString(fmt.Sprintf("| %5d | ", time))
			} else {
				sb.WriteString(fmt.Sprintf("|%01d:%02d.%02d| ", time/3000, (time/50)%60, time%50))
			}

			// Loop for each channel
			for i := 0; i < 3; i++ {
				newnote := 0
        		// Keyoff-keyon sequence detection
				currWave := curr_sid.Channel[i].Wave
				prev2Wave := prev2_sid.Channel[i].Wave
				if (currWave >= 0x10) {
					if ((currWave & 1 == 1) && (((prev2Wave & 1) == 0) || (prev2Wave < 0x10))) {
						prev2_sid.Channel[i].Note = -1
					}
				}

				// Frequency
				if ((frames == opt.Firstframe) || (prev_sid.Channel[i].Note == -1) || (curr_sid.Channel[i].Freq != prev_sid.Channel[i].Freq)) {
					dist := 0x7fffffff
					delta := int(curr_sid.Channel[i].Freq) - int(prev2_sid.Channel[i].Freq)
					sb.WriteString(fmt.Sprintf("%04X ", curr_sid.Channel[i].Freq))

					if (curr_sid.Channel[i].Wave >= 0x10) {
						// Get new note number
						for d := 0; d < 96; d++ {
							cmpfreq := uint16(freqtbllo[d]) | (uint16(freqtblhi[d]) << 8)
							freq := curr_sid.Channel[i].Freq

							if (absInt(int(freq)-int(cmpfreq)) < dist) {
								dist = absInt(int(freq)-int(cmpfreq))
								// favor old note
								if (d == prev_sid.Channel[i].Note) {
									dist /= opt.Oldnotefactor
								}
								curr_sid.Channel[i].Note = d
							}
						}
						
						// Print new note
						curr_note := curr_sid.Channel[i].Note
						prev_note := prev_sid.Channel[i].Note
						if (curr_note != prev_note) {
							if (prev_note == -1) {
								if (opt.Lowres == 1) {
									newnote = 1
								}
								sb.WriteString(fmt.Sprintf("%s %02X  ", notename[curr_note], curr_note | 0x80))
							} else {
								sb.WriteString(fmt.Sprintf("(%s %02X) ", notename[curr_note], curr_note | 0x80))
							}
						} else {
							// If same note, print frequency change (slide/vibrato)
							switch {
								case delta == 0:
									sb.WriteString(" ... ..  ")
								case delta > 0:
									sb.WriteString(fmt.Sprintf("(+ %04X) ", delta))
								case delta < 0:
									sb.WriteString(fmt.Sprintf("(- %04X) ", -delta))
							}
						}	
					} else {
						sb.WriteString(" ... ..  ")
					}
				} else {
					sb.WriteString("....  ... ..  ")
				}

				// Waveform
				if ((frames == opt.Firstframe) || (newnote != 0) || (uint16(curr_sid.Channel[i].Wave) != uint16(prev_sid.Channel[i].Wave))) {
					sb.WriteString(fmt.Sprintf("%02X ", curr_sid.Channel[i].Wave))
				} else {
					sb.WriteString(".. ")
				}

				// ADSR
				if ((frames == opt.Firstframe) || (newnote != 0) || (uint16(curr_sid.Channel[i].ADSR) != uint16(prev_sid.Channel[i].ADSR))) {
					sb.WriteString(fmt.Sprintf("%04X ", curr_sid.Channel[i].ADSR))
				} else {
					sb.WriteString(".... ")
				}				

				// Pulse
				if ((frames == opt.Firstframe) || (newnote != 0) || (uint16(curr_sid.Channel[i].Pulse) != uint16(prev_sid.Channel[i].Pulse))) {
					sb.WriteString(fmt.Sprintf("%03X ", curr_sid.Channel[i].Pulse))
				} else {
					sb.WriteString("... ")
				}				

				sb.WriteString("| ")
			}
		
			// Filter cutoff
			if ((frames == opt.Firstframe) || curr_sid.Filt.Cutoff != prev_sid.Filt.Cutoff) {
				sb.WriteString(fmt.Sprintf("%04X ", curr_sid.Filt.Cutoff))
			} else {
				sb.WriteString(".... ")		
			}

			// Filter control
			if ((frames == opt.Firstframe) || uint16(curr_sid.Filt.Control) != uint16(prev_sid.Filt.Control)) {
				sb.WriteString(fmt.Sprintf("%02X ", curr_sid.Filt.Control))
			} else {
				sb.WriteString(".. ")		
			}

			// Filter passband
			if ((frames == opt.Firstframe) || (uint16(curr_sid.Filt.Type & 0x70) != uint16(prev_sid.Filt.Type & 0x70))) {
				sb.WriteString(fmt.Sprintf("%s ", filtername[(curr_sid.Filt.Type >> 4) & 0x7]))
			} else {
				sb.WriteString("... ")		
			}

			// Mastervolume
			if ((frames == opt.Firstframe) || (uint16(curr_sid.Filt.Type & 0xF) != uint16(prev_sid.Filt.Type & 0xF))) {
				sb.WriteString(fmt.Sprintf("%01X ", curr_sid.Filt.Type & 0xF))
			} else {
				sb.WriteString(". ")		
			}
			
			// Rasterlines / cycle count
			if (opt.Profiling != 0) {
				cycles := cpu.Cycles
				rasterlines := (cycles + 62) / 63
				badlines := ((cycles + 503) / 504)
				rasterlinesbad := (badlines * 40 + cycles + 62) / 63
				sb.WriteString(fmt.Sprintf("| %4d %02X %02X ", cycles, rasterlines, rasterlinesbad))
			}

			// End of frame display, print info so far and copy SID registers to old registers
			sb.WriteString("|\n")

			if ((opt.Lowres != 0) || (((frames - opt.Firstframe) % opt.Spacing) == 0)) {
				fmt.Print(sb.String())
				prev_sid.CopyFrom(curr_sid)
			}
			prev2_sid.CopyFrom(curr_sid)

			// Print note/pattern separators
			if (opt.Spacing != 0) {
				counter++
				if (counter >= opt.Spacing) {
					counter = 0
					if (opt.Pattspacing != 0) {
						rows++
						if (rows >= opt.Pattspacing) {
							rows = 0
							fmt.Printf("+=======+===========================+===========================+===========================+===============+\n")
						} else {
							if (opt.Lowres != 0) {
								fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+\n")
							}
						}
					} else {
						if (opt.Lowres != 0) {
							fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+\n")
						}					
					}
				}
			}
		}

		// Advance to next frame
		frames++
	}

	fmt.Println("Simulation done!")
}

func absInt(x int) int {
	return absDiffInt(x, 0)
 }
 
 func absDiffInt(x, y int) int {
	if x < y {
	   return y - x
	}
	return x - y
 }
