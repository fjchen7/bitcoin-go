package main

// `version` message for checking if current nodes' blockchain is outdated
type version struct {
	Version    int
	BestHeight int    // length of blockchain
	AddrFrom   string // sender's address
}

type address struct {
	AddrList []string
}

type block struct {
	AddrFrom string // sender's address
	Block    []byte
}

type tx struct {
	AddrFrom    string // sender's address
	Transaction []byte
}

// `getblocks` message for getting his list of blocks hashes
type getblocks struct {
	AddrFrom string // sender's address
}

// `getdata` message for getting data
type getdata struct {
	AddrFrom string // sender's address
	Type     string // two type: "tx" or "block"
	ID       []byte // transaction/block hash
}

// `inv` message for showing others nodes what blocks or transactions current node has
// for the purpose of broadcast
type inv struct {
	AddrFrom string   // sender's address
	Type     string   // two type: "tx" or "block"
	Items    [][]byte // don't contain whole blocks or transactions, just their hashes
}
