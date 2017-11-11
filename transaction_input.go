package main

import (
	"bytes"
)

type TXInput struct {
	Txid      []byte // previous transaction id
	Vout      int    // a vout sequence number in previous Txid transaction
	Signature []byte // signature of transaction body
	PubKey    []byte // ScriptSig
}

// checks if `pubKeyHash`(address) initial `in`
func (in *TXInput) UseKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}
