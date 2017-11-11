package main

import (
	"bytes"
	"encoding/binary"
	"log"
)

func logErr(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// convert int into hex
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	logErr(err)
	return buff.Bytes()
}

// ReverseBytes reverses a byte array
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
