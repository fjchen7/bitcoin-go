A simplified Bitcoin implemented in Go, which is good for beginner to learn how Bitcoin work.

-   Original source：[Building Blockchain in Go - Ivan Kuznetsov](https://jeiwan.cc/tags/blockchain/)
-   Usage：`go build` and `./blockchain-go`

### UTXO Set

-   Block are stored in `block` database
-   UTXOs are stored in `chainstate` database

`chainstate` structure

-   `'c' + 32-byte transaction hash -> UTXOs record for that transaction`
-   `'B' -> 32-byte block hash: the block hash up to which the database represents the unspent transaction outputs`

```
Blockchain
	- FindUnspentTransactions([]byte)
	- FindSpendableOutputs
	- FindUTXO
	- FindTransaction
```

Structure in chainstate databse

-   txID->out

