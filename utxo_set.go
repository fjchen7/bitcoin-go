package main

import (
	"encoding/hex"
	"log"

	"github.com/boltdb/bolt"
)

const utxoBucket = "chainstate"

// UTXO set
type UTXOSet struct {
	Blockchain *Blockchain
}

// find a UTXO set that `pubKeyHash` could spend from dabatase
// It won't find all UXTO: if total UXTO amounts is more `amount`, then
// stop finding and return
//
// returns: (total UTXO amounts, txid->[no1, no2, ...])
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}
		return nil
	})
	logErr(err)

	return accumulated, unspentOutputs
}

// find all UTXO that `pubKeyHash` could spend from database
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	logErr(err)

	return UTXOs
}

// return the number of transaction in UTXO set from database
func (u UTXOSet) CountTransactions() int {
	db := u.Blockchain.db
	counter := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}

		return nil
	})
	logErr(err)

	return counter
}

// Rebuild UTXO database: clear UTXO database and build a new one in which
// UTXOs in blockchain (memory) are saved
func (u UTXOSet) Reindex() {
	db := u.Blockchain.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error { // Clear data in database
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		logErr(err)

		return nil
	})
	logErr(err)

	UTXO := u.Blockchain.FindUTXO() // Get UTXO list from blockchain

	err = db.Update(func(tx *bolt.Tx) error { // Save UTXOs into database
		b := tx.Bucket(bucketName)

		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			logErr(err)

			err = b.Put(key, outs.Serialize()) // Structure in database: key->outs.Serialize()
			logErr(err)
		}

		return nil
	})
}

// update UTXO set into database from latest block
// `block` should be the latest block
func (u UTXOSet) Update(block *Block) {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false { // If `tx` is a general tansaction, then we process `tx.Vin`
				for _, vin := range tx.Vin {
					updatedOuts := TXOutputs{}            // `updatedOuts` is the new UTXO to be updated into database
					outsBytes := b.Get(vin.Txid)          // find UTXOs of transaction whose ID is referenced in `vin` from database
					outs := DeserializeOutputs(outsBytes) // `outs`: tTxID->[no1, no2, ...]. Here tx

					for outIdx, out := range outs.Outputs { // `outIdx` is index of output `out` in transaction which referenced in `vin`
						if outIdx != vin.Vout {
							// `outIdx`==vin.Vout means that output `out` has
							// been spent, and we don't add it into updated UTXO
							// set
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					// Here `updatedOuts` is all UTXOs in transaction whose ID
					// is `vin.Txid`
					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						logErr(err)
					} else {
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						logErr(err)
					}

				}
			}

			// In latest block `block`, all outputs in each transactions are
			// UTXOs.
			newOutputs := TXOutputs{}
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			err := b.Put(tx.ID, newOutputs.Serialize())
			logErr(err)
		}

		return nil
	})
	logErr(err)
}
