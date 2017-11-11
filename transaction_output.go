package main

import (
	"bytes"
)

type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

// Let `address` lock `output`
func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

// check if `pubKeyHash` could unlock `out`
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// create a new TXOutput
func NewTXOutput(value int, addr string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(addr))

	return txo
}
