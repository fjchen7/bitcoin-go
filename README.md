A simplified Bitcoin implemented in Go, which is good for beginner to learn how Bitcoin work.

-   From
    -   tutorial [Building Blockchain in Go - Ivan Kuznetsov](https://jeiwan.cc/tags/blockchain/)
    -   repository [Jeiwan/blockchain_go](https://github.com/Jeiwan/blockchain_go)
-   Usage：`go build` and `./blockchain-go`

## Transaction

### UTXO Set

-   Block are stored in `block` database
-   UTXOs are stored in `chainstate` database

`chainstate` structure

-   `'c' + 32-byte transaction hash -> UTXOs record for that transaction`
-   `'B' -> 32-byte block hash: the block hash up to which the database represents the unspent transaction outputs`


## Network

In Bitcoin Core, there are [DNS seeds](https://bitcoin.org/en/glossary/dns-seed) hardcoded which help node find other nodes to connect Bitcoin network for the first time.

We have three nodes:

1.  Central node which all nodes will connect to.
2.  Miner node which will store transactions in mempool and mine blocks.
3.  Wallet node which will be used to send coins between wallets. Unlike SPV nodes though, it’ll store a full copy of blockchain.
