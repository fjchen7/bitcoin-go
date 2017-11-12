// main.go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println(os.Args)
	cli := CLI{}
	if len(os.Args) == 1 {
		cli.usage()
	} else {
		cli.handleCommands(os.Args[1:])
	}

}
