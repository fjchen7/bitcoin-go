package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

// sendVersion sends `addr` `version` message
func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(version{
		nodeVersion,
		bestHeight,
		nodeAddr,
	})

	// Message are sequence of bytes on low level.
	// First 12 bytes speficy command ("version" here) and latter bytes are
	// gob-encoded message structure
	request := append(commandToBytes("version"), payload...)

	sendData(addr, request)
}

// send `data` to `addr`
func sendData(addr string, data []byte) {
	// connect to `addr`
	conn, err := net.Dial(protocol, addr)

	// if connect error
	if err != nil {
		fmt.Println("%s is not available\n", addr)
		var updatedNodes []string

		// update `knownNodes`
		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes // new `knownNodes` that don't contain `addr`

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	logErr(err)
}

func sendAddr(addr string) {
	nodes := address{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddr)
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)

	sendData(addr, request)
}

func sendBlock(addr string, b *Block) {
	data := block{nodeAddr, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	sendData(addr, request)
}

func sendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddr, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(addr, request)
}

// send `inv` message to `addr`
func sendInv(addr, kind string, items [][]byte) {
	inventory := inv{nodeAddr, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	sendData(addr, request)
}

// send `getblocks` message to `addr`
func sendGetBlocks(addr string) {
	payload := gobEncode(getblocks{nodeAddr})
	request := append(commandToBytes("getblocks"), payload...)

	sendData(addr, request)
}

// send a message to `addr` for getting
func sendGetData(addr, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddr, kind, id})
	request := append(commandToBytes("getdata"), payload...)

	sendData(addr, request)
}
