package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
)

type Wallets struct {
	Wallets map[string]*Wallet
}

const (
	walletFile = "wallet.dat"
)

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}

	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFromFile()

	return &wallets, err
}

// add a Wallet to `ws`
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	addr := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[addr] = wallet

	return addr
}

// return all addresses stored in *ws*
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for addr := range ws.Wallets {
		addresses = append(addresses, addr)
	}

	return addresses
}

// return a Wallet by addr
func (ws Wallets) GetWallet(addr string) Wallet {
	return *ws.Wallets[addr]
}

// loads Wallets from data file
func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	logErr(err)

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	logErr(err)

	ws.Wallets = wallets.Wallets

	return nil
}

func (ws Wallets) SaveToFile() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	logErr(err)

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	logErr(err)

}
