package main

// Filter represents the filter in the SID chip.
type Filter struct {
	Type    uint8
	Control uint8
	Cutoff  uint16
}

// Init initializes all properties.
func (f *Filter) Init() {
    f.Type = 0;
    f.Control = 0;
    f.Cutoff = 0;
}

func (f *Filter) CopyFrom(src *Filter) {
    f.Type = src.Type
    f.Control = src.Control
    f.Cutoff = src.Cutoff
}