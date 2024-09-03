package main

import (
	"fmt"
	"strings"
)

// define a commmon interface for all output decoders
type SidOutputDecoder interface {
	PreSteps()
	ProcessFrame(frame int, cycles uint64)
	PostSteps()
}

type ActiveDecoder struct {
    decoder SidOutputDecoder
}

func (d *ActiveDecoder) SetOutput(dec SidOutputDecoder) {
    d.decoder = dec
}

func (d *ActiveDecoder) PreProcess() {
    d.decoder.PreSteps()
}

func (d *ActiveDecoder) ProcessFrame(frame int, cycles uint64) {
    d.decoder.ProcessFrame(frame, cycles)
}

func (d *ActiveDecoder) PostProcess() {
    d.decoder.PostSteps()
}

type ScreenOutputWithNotes struct {
	Options        *SidOutputSettings
	SidState       *Sid

	prevSidState [2]*Sid
	counter      int
	rows         int
}

// use struct to implement interface
func (state *ScreenOutputWithNotes) PreSteps() {
	state.prevSidState[0] = NewSID()
	state.prevSidState[1] = NewSID()

	fmt.Printf("Middle C frequency is $%04X\n\n", uint16(freqtbllo[48])|(uint16(freqtblhi[48])<<8))
	fmt.Printf("| Frame | Freq Note/Abs WF ADSR Pul | Freq Note/Abs WF ADSR Pul | Freq Note/Abs WF ADSR Pul | FCut RC Typ V |")

	if state.Options.Profiling != 0 {
		// CPU cycles, Raster lines, Raster lines with badlines on every 8th line, first line included
		fmt.Printf(" Cycl RL RB |")
	}
	fmt.Printf("\n")
	fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+")
	if state.Options.Profiling != 0 {
		fmt.Printf("------------+")
	}
	fmt.Printf("\n")
}

func (state *ScreenOutputWithNotes) ProcessFrame(frame int, cycles uint64) {
	var sb strings.Builder

	opt := state.Options
	currentSid := state.SidState
	prev2Sid := state.prevSidState[1]
	prevSid := state.prevSidState[0]

	time := frame - opt.Firstframe
	firstframe := (frame == opt.Firstframe)

	if opt.Timeseconds == 0 {
		sb.WriteString(fmt.Sprintf("| %5d | ", time))
	} else {
		sb.WriteString(fmt.Sprintf("|%01d:%02d.%02d| ", time/3000, (time/50)%60, time%50))
	}
	// Loop for each channel
	for i := 0; i < 3; i++ {
		newnote := false
		// Keyoff-keyon sequence detection
		currWave := currentSid.Channel[i].Wave
		prev2Wave := prev2Sid.Channel[i].Wave
		if currWave >= 0x10 {
			if (currWave&1 == 1) && (((prev2Wave & 1) == 0) || (prev2Wave < 0x10)) {
				prev2Sid.Channel[i].Note = -1
			}
		}

		// Frequency
		if (firstframe) || (prevSid.Channel[i].Note == -1) || (currentSid.Channel[i].Freq != prevSid.Channel[i].Freq) {
			dist := 0x7fffffff
			delta := int(currentSid.Channel[i].Freq) - int(prev2Sid.Channel[i].Freq)
			sb.WriteString(fmt.Sprintf("%04X ", currentSid.Channel[i].Freq))

			if currentSid.Channel[i].Wave >= 0x10 {
				// Get new note number
				for d := 0; d < 96; d++ {
					cmpfreq := uint16(freqtbllo[d]) | (uint16(freqtblhi[d]) << 8)
					freq := currentSid.Channel[i].Freq

					if absInt(int(freq)-int(cmpfreq)) < dist {
						dist = absInt(int(freq) - int(cmpfreq))
						// favor old note
						if d == prevSid.Channel[i].Note {
							dist /= opt.Oldnotefactor
						}
						currentSid.Channel[i].Note = d
					}
				}

				// Print new note
				curr_note := currentSid.Channel[i].Note
				prev_note := prevSid.Channel[i].Note
				if curr_note != prev_note {
					if prev_note == -1 {
						if opt.Lowres == 1 {
							newnote = true
						}
						sb.WriteString(fmt.Sprintf("%s %02X  ", notename[curr_note], curr_note|0x80))
					} else {
						sb.WriteString(fmt.Sprintf("(%s %02X) ", notename[curr_note], curr_note|0x80))
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
		if (firstframe) || (newnote) || (uint16(currentSid.Channel[i].Wave) != uint16(prevSid.Channel[i].Wave)) {
			sb.WriteString(fmt.Sprintf("%02X ", currentSid.Channel[i].Wave))
		} else {
			sb.WriteString(".. ")
		}

		// ADSR
		if (firstframe) || (newnote) || (uint16(currentSid.Channel[i].ADSR) != uint16(prevSid.Channel[i].ADSR)) {
			sb.WriteString(fmt.Sprintf("%04X ", currentSid.Channel[i].ADSR))
		} else {
			sb.WriteString(".... ")
		}

		// Pulse
		if (firstframe) || (newnote) || (uint16(currentSid.Channel[i].Pulse) != uint16(prevSid.Channel[i].Pulse)) {
			sb.WriteString(fmt.Sprintf("%03X ", currentSid.Channel[i].Pulse))
		} else {
			sb.WriteString("... ")
		}

		sb.WriteString("| ")
	}

	// Filter cutoff
	if (firstframe) || currentSid.Filt.Cutoff != prevSid.Filt.Cutoff {
		sb.WriteString(fmt.Sprintf("%04X ", currentSid.Filt.Cutoff))
	} else {
		sb.WriteString(".... ")
	}

	// Filter control
	if (firstframe) || uint16(currentSid.Filt.Control) != uint16(prevSid.Filt.Control) {
		sb.WriteString(fmt.Sprintf("%02X ", currentSid.Filt.Control))
	} else {
		sb.WriteString(".. ")
	}

	// Filter passband
	if (firstframe) || (uint16(currentSid.Filt.Type&0x70) != uint16(prevSid.Filt.Type&0x70)) {
		sb.WriteString(fmt.Sprintf("%s ", filtername[(currentSid.Filt.Type>>4)&0x7]))
	} else {
		sb.WriteString("... ")
	}

	// Mastervolume
	if (firstframe) || (uint16(currentSid.Filt.Type&0xF) != uint16(prevSid.Filt.Type&0xF)) {
		sb.WriteString(fmt.Sprintf("%01X ", currentSid.Filt.Type&0xF))
	} else {
		sb.WriteString(". ")
	}

	// Rasterlines / cycle count
	if opt.Profiling != 0 {
		// cycles := cpu.Cycles
		//cycles := state.CyclesForFrame
		rasterlines := (cycles + 62) / 63
		badlines := ((cycles + 503) / 504)
		rasterlinesbad := (badlines*40 + cycles + 62) / 63
		sb.WriteString(fmt.Sprintf("| %4d %02X %02X ", cycles, rasterlines, rasterlinesbad))
	}

	// End of frame display, print info so far and copy SID registers to old registers
	sb.WriteString("|\n")

	if (opt.Lowres != 0) || (((time) % opt.Spacing) == 0) {
		fmt.Print(sb.String())
		prevSid.CopyFrom(currentSid)
	}
	prev2Sid.CopyFrom(currentSid)

	// Print note/pattern separators
	if opt.Spacing != 0 {
		state.counter++
		if state.counter >= opt.Spacing {
			state.counter = 0
			if opt.Pattspacing != 0 {
				state.rows++
				if state.rows >= opt.Pattspacing {
					state.rows = 0
					fmt.Printf("+=======+===========================+===========================+===========================+===============+\n")
				} else {
					if opt.Lowres != 0 {
						fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+\n")
					}
				}
			} else {
				if opt.Lowres != 0 {
					fmt.Printf("+-------+---------------------------+---------------------------+---------------------------+---------------+\n")
				}
			}
		}
	}
}

func (state *ScreenOutputWithNotes) PostSteps() {}

// struct to implement decoder for screen output with notes
// info.
type ScreenOutputSidRegisters struct {
	Options        *SidOutputSettings
	SidState       *Sid

	prevSidState *Sid
	// counter      int
	// rows         int
}

func (state *ScreenOutputSidRegisters) PreSteps() {
	state.prevSidState = NewSID()
	fmt.Printf("| Frame | 00 01 02 03 04 05 06 | 07 08 09 10 11 12 13 | 14 15 16 17 18 19 20 | 21 22 23 24 | dt_us |");
	fmt.Printf("\n");
	fmt.Printf("+-------+----+-----------------+----------------------+----------------------+-------------+-------+");
	fmt.Printf("\n");
}
func (state *ScreenOutputSidRegisters) ProcessFrame(frame int, cycles uint64) {
	var sb strings.Builder

	opt := state.Options
	currentSid := state.SidState
	prevSid := state.prevSidState
	time := frame - opt.Firstframe

	if opt.Timeseconds == 0 {
		sb.WriteString(fmt.Sprintf("| %5d | ", time))
	} else {
		sb.WriteString(fmt.Sprintf("|%01d:%02d.%02d| ", time/3000, (time/50)%60, time%50))
	}

	// Check registers for changes, print the ones that have changed
	for c := 0; c < 25; c++ {
		if ((currentSid.Register[c] != prevSid.Register[c]) || (time == 0)) {
			sb.WriteString(fmt.Sprintf("%02X ", currentSid.Register[c]))
		} else {
			sb.WriteString(".. ")
		}

		if (c==6 || c==13 || c==20) {
			sb.WriteString("| ")	
		}

		prevSid.Register[c] = currentSid.Register[c];
	} 
	sb.WriteString(fmt.Sprintf("|  %04X ", (uint16(currentSid.Register[25]) << 8) | uint16(currentSid.Register[26])))
	sb.WriteString("|\n");
	fmt.Print(sb.String())
}

func (state *ScreenOutputSidRegisters) PostSteps() {}
