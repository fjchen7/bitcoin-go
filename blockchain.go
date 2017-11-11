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
func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var lastHash []byte

	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) != true {
			log.Println("ERROR: Invalid transaction when mining")
			return
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
		//b := tx.Bucket([]byte(blocksBucket))

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

// return an iterator for reading block
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

// return true if db file exisit, otherwise false
func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// find transaction whose tx.ID equals to `ID`
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

// returns transactions set that contain pubKeyHash's UTXO.
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int) /* txid->[no1, no2, ...]
	it means txid Transaction's Number no1、no2... TXOutputs
	have been spent by `pubKeyHash` */
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							/* `spendOut`==`outIdx` means that the TXOutput
							has been spent by `pubKeyHash` */
							continue Outputs // jump to next tx.Vout
						}
					}
				}
				/* If program run here, it means that `tx` has TXOutput which
				`pubKeyHash` can spend.
				But there needs a prerequisite: `tx` has 1 output which `pubKeyHash`
				can spend at most, otherwise `tx` will be added many time into
				unspentTXs set */
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}
			if tx.IsCoinbase() == false { // If `tx` is not coinbase
				for _, in := range tx.Vin {
					if in.UseKey(pubKeyHash) {
						/* If this txin has spent `pubKeyHash`'s outputs.
						it implies `pubKeyHash` create the txin and spent it. */
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return unspentTXs
}

// return all UTXO that `pubKeyHash` can spend
func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// find UTXO set `pubKeyHash` could spend.
// It won't find all UXTO: if total UXTO amounts is more `amount`, then
// stop finding
//
// returned map: (total UTXO amounts, txid->[no1, no2, ...])
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0
Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOutputs
}

// sign `tx` by `privKey`
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
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		logErr(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
