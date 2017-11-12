// main.go
package main

import (
	"fmt"
	"os"
	// "fmt"
	//"strconv"
)

func main() {
	/* Part2
	bc := NewBlockchain()
	bc.AddBlock("Send 1 BTC to fjchen")
	bc.AddBlock("Send 2 more BTC to fjchen")
	for _, block := range bc.blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n",
			strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
	*/
	/* Part3
	bc := NewBlockchain()
	defer bc.db.Close()
	cli := CLI{bc}
	cli.Run()
	*/
	/* Part4 Test
	tx := Transaction{nil, []TXInput{}, []TXOutput{}}
	fmt.Println(tx)
	tx.SetID()
	fmt.Println(tx)
	*/
	// Part4
	fmt.Println(os.Args)
	cli := CLI{}
	cli.handleCommands(os.Args[1:])
	//cli.init()

}
