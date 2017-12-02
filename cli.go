// cli.go
package main

import (
	"fmt"
	"log"
	"strconv"
)

type CLI struct{}

// user manual
func (cli CLI) usage() {
	fmt.Println(`
		createblockchain <address>  --  Create a blockchain and send genesis block reward to <address>
		createwallet  --  Generates a new key-pair and saves it into the wallet file"
		chain  --  Print all blocks of the blockchain
		address  --  List all addresses from the wallet file
		balance <address>   --  Get balance of <address>
		send <from> <to> <amount>  -- Send <amount> of coins from <from> to <to>
			`)
}

/*
// initial CLI tool
func (cli CLI) init() {
	cli.usage()
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command-> ")
		rawLine, _, _ := r.ReadLine()
		line := string(rawLine)
		if line == "q" || line == "quit" {
			break
		}
		tokens := strings.Split(line, " ")
		cli.handleCommands(tokens)
	}

}
*/

// handle command arguments
func (cli CLI) handleCommands(tokens []string) {
	switch tokens[0] {
	case "createblockchain":
		if len(tokens) == 2 {
			addr := tokens[1]
			cli.createBlockchain(addr)
		} else {
			fmt.Println("USAGE: creatleblockchain <address>")
		}
	case "createwallet":
		cli.createWallet()
	case "chain":
		cli.printChain()
	case "address":
		cli.listAddresses()
	case "balance":
		if len(tokens) == 2 {
			addr := tokens[1]
			cli.getBalance(addr)
		} else {
			fmt.Println("USAGE: balance <address>")
		}
	case "send":
		if len(tokens) == 4 {
			from := tokens[1]
			to := tokens[2]
			amount, err := strconv.Atoi(tokens[3])
			if err == nil {
				cli.send(from, to, amount)
			} else {
				fmt.Println("USAGE: send <from> <to> <amount>")
			}
		} else {
			fmt.Println("USAGE: send <from> <to> <amount>")
		}
	default:
		cli.usage()
	}
}

// print chain
func (cli *CLI) printChain() {
	bc := LoadBlockchain()
	defer bc.db.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}
func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	addr := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("You new address: %s\n", addr)
}

func (cli *CLI) listAddresses() {
	wallets, err := NewWallets()
	logErr(err)

	addresses := wallets.GetAddresses()

	for _, addr := range addresses {
		fmt.Println(addr)
	}
}

// get addr's balance
func (cli *CLI) getBalance(addr string) {
	if !ValidateAddress(addr) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := LoadBlockchain()
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)
	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of '%s': %d\n", addr, balance)
}

// send `amount` from `from` to `to`
func (cli *CLI) send(from, to string, amount int) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}
	bc := LoadBlockchain()
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	tx := NewUTXOTransaction(from, to, amount, &UTXOSet)
	cbTx := NewCoinbaseTX(from, "")
	txs := []*Transaction{cbTx, tx}

	newBlock := bc.MineBlock(txs) // the mined block only contains a coinbase and transaction which `from` send `to`
	UTXOSet.Update(newBlock)      //update UTXO database
	if newBlock != nil {
		fmt.Println("Send Success!")
	} else {
		fmt.Println("Send Failed, Not Enough Amounts!")
	}
}

// create a new blockchain
func (cli *CLI) createBlockchain(addr string) {
	if !ValidateAddress(addr) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(addr)
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	fmt.Println("Create Blockchain Success!")
}
