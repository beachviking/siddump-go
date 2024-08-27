package main

type Sid struct {
	Channel [3]Voice
	Filt    Filter
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
}