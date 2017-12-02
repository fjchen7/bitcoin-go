package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var nodeAddr string

var miningAddress string                    // the mining reward payee
var knownNodes = []string{"localhost:3000"} // central node
var blocksInTransit = [][]byte{}            // a block hash set waiting to be downloaded
var mempool = make(map[string]Transaction)

// StartServer starts a node to connect network
// `nodeID`: constructing IP address "localhost:`nodeID`"
// `minerAddress` : the address to received mining rewards to
func StartServer(nodeID, minerAddress string) {
	nodeAddr = fmt.Sprintf("localhost:%s", nodeID)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddr)
	logErr(err)
	defer ln.Close()

	bc := NewBlockchain(nodeID)

	// If current node is not central node, then send `version` message to
	// central node to know if its blockchain is outdated
	if nodeAddr != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		//logErr(err)
		go handleConnection(conn, bc)
	}
}

// commandToBytes converts `command` into a 12-byte buffer
func commandToBytes(command string) []byte {
	var bytes [commandLength]byte
	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// bytesToCommand converts 12-byte buffer `bytes` into command string
func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
