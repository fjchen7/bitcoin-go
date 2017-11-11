package main

import (
	"github.com/boltdb/bolt"
	"log"
)

/* 区块存在按 byte-sorted 顺序存在 BoltDB 数据库里。
   我们想把区块按序打印出来，又不想把所有的区块都加载到内存里。
   于是我们要一个一个地读取区块，而 BlockchainIterator 就是做这个事 */
type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

// 返回区块链里的下一个区块
func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash // 指向前一个区块

	return block
}
