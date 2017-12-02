package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 10

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

// check if the transaction is coinbase
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// serialize `tx`
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	logErr(err)

	return encoded.Bytes()
}

// serialize `tx` and hash it with SHA-256 algorithm.
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]

}

// return a transaction which empties Sinature and PubKey filed in all TXInpus
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput
	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}
	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}
	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

// signs each input of `tx`
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey,
	prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}
	txCopy := tx.TrimmedCopy()
	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)] // previous transactions
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()     //? SegWit: txid doesn't include signature
		txCopy.Vin[inID].PubKey = nil // what does this mean?
		// tutorial says "After getting the hash we should reset the PubKey
		// field, so it doesn’t affect further iterations."

		// Signing process
		// 1. Empty Signature and PubKey filed in all TXInputs
		// 2. Fill PubKey field in corresponding TXInput
		// 3. Hash whole transaction
		// 4. Sign the hash value
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		logErr(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[inID].Signature = signature
	}
}

// check if Pubkey in `tx` TXInputs could verify
// Signature in transaction TXOutputs from `prevTXs`
// prevTXs structure: Transaction.ID->Transaction
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() { // coinbase transaction don't need verification
		return true
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)] // previous transactions
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		// signature (r, s) is a pair of numbers
		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		// public key (x, y) is a pair of coordinates
		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}
	return true
}

// create a new coinbase transaction
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	txin := TXInput{[]byte{}, -1, nil, []byte(data)} // coinbase have an empty TXInput
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

// create a general transaction
func NewUTXOTransaction(from, to string, amount int, UTXOSet *UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := NewWallets() // load wallets
	logErr(err)
	wallet := wallets.GetWallet(from) // 1. load wallet by address `from`
	pubKeyHash := HashPubKey(wallet.PublicKey)
	// 2. find UTXO that address `from` can spend
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Print("ERROR: Not enough funds")
		return nil
	}

	// 3. construct TXInputs
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid) // decode UTXO's txid into []byte
		logErr(err)

		for _, out := range outs {
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// 4. construct TXOutputs
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount { // change（找零）
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx

}

// return a human-readable representation of a transaction
func (tx Transaction) String() string {
	var lines []string

	if tx.IsCoinbase() {
		lines = append(lines, fmt.Sprintf("---   Coinbase  %x:", tx.ID))
		lines = append(lines, fmt.Sprintf("       Data:    %s:", tx.Vin[0].PubKey))

	} else {
		lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

		for i, input := range tx.Vin {

			lines = append(lines, fmt.Sprintf("     Input %d:", i))
			lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
			lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
			lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
			lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
		}
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// DeserializeTransaction deserializes a transaction
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}
