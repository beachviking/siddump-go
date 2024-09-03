package main

import "flag"

type SidOutputSettings struct {
	Basefreq      int
	Basenote      int
	Firstframe    int
	Lowres        int
	Oldnotefactor int
	Pattspacing   int
	Profiling     int
	Subtune       int
	Seconds       int
	Spacing       int
	Timeseconds   int
	Usage         int
	DecoderOutput int
}

func (opt *SidOutputSettings) ParseArgs() {
	flag.IntVar(&opt.Subtune, "a", 0, "Accumulator value on init (subtune number) default = 0")
	flag.IntVar(&opt.Basefreq, "c", 0, "Frequency recalibration. Give note frequency in hex")
	flag.IntVar(&opt.Basenote, "d", 0xb0, "Select calibration note (abs.notation 80-DF). Default middle-C (B0)")
	flag.IntVar(&opt.Firstframe, "f", 0, "First frame to display, default 0")
	flag.IntVar(&opt.Lowres, "l", 1, "Low-resolution mode (only display 1 row per note)")
	flag.IntVar(&opt.DecoderOutput, "m", 0, "Output mode, default 0")
	flag.IntVar(&opt.Spacing, "n", 0, "Note spacing, default 0 (none)")
	flag.IntVar(&opt.Oldnotefactor, "o", 1, "'Oldnote-sticky' factor. Default 1, increase for better vibrato display")
	flag.IntVar(&opt.Pattspacing, "p", 0, "Pattern spacing, default 0 (none)")
	flag.IntVar(&opt.Timeseconds, "s", 0, "Display time in minutes:seconds:frame format")
	flag.IntVar(&opt.Seconds, "t", 60, "Playback time in seconds, default 60")
	flag.IntVar(&opt.Usage, "h", 0, "Display usage information")
	flag.IntVar(&opt.Profiling, "z", 0, "Include CPU cycles+rastertime (PAL)+rastertime, badline corrected")
	flag.Parse()	
}