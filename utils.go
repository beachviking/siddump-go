package main

import (
	"encoding/binary"
	"os"
)

func readByte(f *os.File) byte {
	var res byte
	err := binary.Read(f, binary.LittleEndian, &res)
	check(err)
	return res
}

func readWord(f *os.File) uint16 {
	var res [2]byte
	err := binary.Read(f, binary.LittleEndian, &res)
	check(err)
	word := uint16(res[0])<<8 | uint16(res[1])
	return word
}

func check(e error) {
	if e != nil {
		panic(e)
	}
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
