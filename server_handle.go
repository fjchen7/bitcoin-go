package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
)

// handleConnection handles different commands contained in `conn`
func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)
	logErr(err)

	// extract command
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

/* ---------- Functions below are handling different message ---------- */

func handleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	//logErr(err)

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		// If connected node's blockchain is longer, it replies `getblocks` message
		// to sender of `version` message for getting newer blocks
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		// If current node's blockchain is longer, it replies with `version` message
		sendVersion(payload.AddrFrom, bc)
	}

}

func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload address

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	// update `knownNodes`
	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Println("The are %d known nodes now!\n", len(knownNodes))
	requestBlocks()

}

func requestBlocks() {
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

// When received a `block` message, add it if
func handleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	fmt.Println("Received a new block!")
	bc.AddBlock(block)

	fmt.Println("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		// each `handleBlock` only download one block
		blockHash := blocksInTransit[0]
		// send `getdata` message for getting block whose hash is `blockHash`
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}

}

func handleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	fmt.Println("Received inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		// Here we only deal with `.Items[0]` because we won't send `inv` with multiple items
		blockHash := payload.Items[0]

		// send `getdata` message for getting block whose hash is `blockHash`
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		// check if tx hash is alread in our mempool.
		// If not, send `getdata` message for getting transaction whose hash is `txID`
		if mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

func handleGetBlocks(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	blocks := bc.GetBlockHashes()

	// reply `inv` message for showing others nodes `payload.AddrFrom`
	// what blocks or transactions he has
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleGetData(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		logErr(err)

		// reply with block data
		sendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		// reply with transaction data
		sendTx(payload.AddrFrom, &tx)
	}
}

func handleTx(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	logErr(err)

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	if nodeAddr == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddr && node != payload.AddrFrom {
				// show to `node` transaction `tx`
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			// Begin to mine block
			var txs []*Transaction

			for id := range mempool {
				tx := mempool[id]
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			cbTx := NewCoinbaseTX(miningAddress, "")
			txs = append(txs, cbTx)

			newBlock := bc.MineBlock(txs)
			UTXOSet := UTXOSet{bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")

			// update `mempool`
			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			// broadcast block
			for _, node := range knownNodes {
				if node != nodeAddr {
					sendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}
