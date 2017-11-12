package main

import (
	"bytes"
	"encoding/gob"
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

type TXOutputs struct {
	Outputs []TXOutput
}

// serialize TXOutputs
func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	logErr(err)

	return buff.Bytes()
}

// deserialize TXOutputs
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	logErr(err)

	return outputs
}
