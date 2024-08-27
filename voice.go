package main

// Voice represents a voice in the SID chip.
type Voice struct {
    Freq    uint16
    Pulse   uint16
    ADSR    uint16
    Wave    uint8
    Note    int
}

// Init initializes all properties.
func (v *Voice) Init() {
    v.Freq = 0;
    v.Pulse = 0;
    v.ADSR = 0;
    v.Wave = 0;
    v.Note = 0;
}

func (v *Voice) CopyFrom(src *Voice) {
    v.Freq = src.Freq
    v.Pulse = src.Pulse
    v.ADSR = src.ADSR
    v.Wave = src.Wave
    v.Note = src.Note
}