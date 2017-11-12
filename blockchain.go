// blockchain.go
package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const (
	dbFile              = "blockchain.db"
	blocksBucket        = "blocks"
	genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
)

type Blockchain struct {
	tip []byte // latest block hash
	db  *bolt.DB
}

// mine a new block which contains `transactions`
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte

	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) != true {
			log.Println("ERROR: Invalid transaction when mining")
			return nil
		}
	}

	err := bc.db.View(func(tx *bolt.Tx) error { // get latest block from database. This is a read-only transaction.
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	logErr(err)

	newBlock := NewBlock(transactions, lastHash)

	err = bc.db.Update(func(tx *bolt.Tx) error { // add a new block into database
		b := tx.Bucket([]byte(blocksBucket))
		err = b.Put(newBlock.Hash, newBlock.Serialize())
		logErr(err)

		err = b.Put([]byte("l"), newBlock.Hash)
		logErr(err)

		bc.tip = newBlock.Hash

		return nil
	})
	logErr(err)

	return newBlock
}

// get Blockchain instance from database
func LoadBlockchain() *Blockchain {
	if dbExists() == false {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	logErr(err)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})
	logErr(err)

	bc := Blockchain{tip, db}

	return &bc
}

// Create a new blockchain and send genesis block reward to `addr`
func CreateBlockchain(addr string) *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte // latest block hash

	cbtx := NewCoinbaseTX(addr, genesisCoinbaseData)
	genesis := NewGenesisBlock(cbtx)

	db, err := bolt.Open(dbFile, 0600, nil) // open BoltDB database file
	logErr(err)

	err = db.Update(func(tx *bolt.Tx) error { // BobtDB has two kinds of transaction（事务）: read-only and read-write. Here we open a read-write transaction.
		b, err := tx.CreateBucket([]byte(blocksBucket))
		logErr(err)

		err = b.Put(genesis.Hash, genesis.Serialize())
		logErr(err)

		err = b.Put([]byte("l"), genesis.Hash)
		logErr(err)

		tip = genesis.Hash

		return nil
	})
	logErr(err)

	bc := Blockchain{tip, db}
	return &bc
}

// find transaction by tx.ID
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction is not found")
}

// find all unspent transaction outputs and returns transactions with only unspent outputs
func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)  //TxID->[output1, output2,...]
	spentTXOs := make(map[string][]int) //TxID->[no1, no2, ...]
	bci := bc.Iterator()

	for {
		block := bci.Next() // process order: latest -> older

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID) // ID of transaction 'tx'

		Outputs:
			for outIdx, out := range tx.Vout { // `outIdx` is index of output `out` in transaction `tx`
				// Was the output spent?
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							// `spendOutIdx`==`outIdx` means that output `out` which
							// index is `outIdx` in transaction `tx` has been
							// spent.
							continue Outputs // We check the next output in transaction `tx`
						}
					}
				}

				// If it comes to here, then output `out` in transaction `tx` is
				// unspent, and it should be added into UTXO set.
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out) // add `out` to `UXTO[txID]`
				UTXO[txID] = outs
			}

			if tx.IsCoinbase() == false {
				// if `tx` is not coinbase, we should record those outputs
				// referenced in `tx.Vin` as spent
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

// return an iterator for reading block
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
} // sign `tx` by `privKey`
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		logErr(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	tx.Sign(privKey, prevTXs)
}

// Check if `tx` could be verified by old transactions in blockchain
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {

	if tx.IsCoinbase() {
		return true
	}
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		logErr(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

// return true if db file exisit, otherwise false
func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}
