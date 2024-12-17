package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Block struct {
	Height     int         `json:"height"`
	TxData     Transaction `json:"transaction_data"`
	Timestamps string      `json:"timestamps"`
	Nonce      int         `json:"nonce"`
	Hash       string      `json:"hash"`
	PrevHash   string      `json:"prev_hash"`
}

type Blockchain struct {
	Chain      []*Block
	Difficulty int
	mu         sync.Mutex
}

var bc Blockchain

var forks []Blockchain

func NewBlockchain() *Blockchain {
	genesisBlock := &Block{}
	genesisBlock = &Block{
		Hash: genesisBlock.calculateHash(),
	}

	bc.Chain = append(bc.Chain, genesisBlock)

	return &bc
}

func NewBlock(tx Transaction) *Block {
	var blockHeight int
	var pvHash string

	if len(bc.Chain) == 0 {
		blockHeight = 0
		pvHash = ""
	} else {
		blockHeight = bc.Chain[len(bc.Chain)-1].Height + 1
		pvHash = bc.Chain[len(bc.Chain)-1].Hash
	}

	b := Block{
		Height:     blockHeight,
		TxData:     tx,
		Timestamps: time.Now().String(),
		PrevHash:   pvHash,
	}
	return &b
}

func MineBlocks(bReceiver <-chan *Block, publisherTopic *pubsub.Topic) {

	for {
		tx := &Transaction{TxID: "null"}
		for _, t := range mempool.transactions {
			tx = t
			break
		}

		b := NewBlock(*tx)
		spew.Dump(b)
		fmt.Println(Reset)

		logger.Info("Mining for new block...")
	NonceFinder:
		for i := 0; ; i++ {
			b.Nonce = i
			hash := b.calculateHash()
			prefix := strings.Repeat("0", 2)

			if strings.HasPrefix(hash, prefix) {
				b.Hash = hash
				logger.Successf("Hell yeah!! Block is mined: %s \n", hash)

				if !b.isValid() {
					logger.Warn("Skipping to add block: Invalid block generated by miner")
					continue
				}

				bc.AddBlock(b)
				blockBytes, err := json.Marshal(b)

				if err != nil {
					logger.Errorf("Invalid json structure of block generated by miner: %v\n", err)
				}

				err = publisherTopic.Publish(context.Background(), blockBytes)

				if err != nil {
					logger.Errorf("Error publishing block: %v\n", err)
				} else {
					logger.Successf("Block: %s broadcasted successfully \n!", b.Hash)
				}
				break NonceFinder
			}

			select {
			// In case new block is received we check if the its index
			// if received blocks'index == current minging block index -> stop mining
			case receivedBlock := <-bReceiver:
				logger.Infof("Block is received %s", b.Hash)
				if receivedBlock.Height == b.Height {
					break NonceFinder
				}
			default:
			}

			time.Sleep(time.Millisecond * 400)
		}

	}
}

func (bc *Blockchain) AddBlock(b *Block) {
	bc.mu.Lock()
	bc.Chain = append(bc.Chain, b)
	bc.mu.Unlock()
}

func (b *Block) isValid() bool {
	if len(bc.Chain) > 0 {
		prevBlock := bc.Chain[len(bc.Chain)-1]
		if prevBlock.Height+1 != b.Height {
			logger.Error("Block validation failed: invalid block height")
			return false
		}
		if prevBlock.Hash != b.PrevHash {
			logger.Error("Block validation failed: previous block hash not matched")
			return false
		}
	}
	if !b.validateHash() {
		logger.Error("Block validation failed: invalid hash")
		return false
	}

	return true
}

func (b *Block) isNewFork() bool {
	if b.Height == bc.Chain[len(bc.Chain)-1].Height {
		return true
	}
	return false
}

func (b *Block) validateHash() bool {
	h := b.calculateHash()
	if h != b.Hash {
		return false
	}
	return true
}

func (b *Block) calculateHash() string {
	data := strconv.Itoa(b.Height) + b.TxData.String() + b.Timestamps + strconv.Itoa(b.Nonce) + b.PrevHash

	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

func (bc *Blockchain) GetLatestBlockHeight() int {
	if len(bc.Chain) == 0 {
		return -1
	}
	blockHeight := bc.Chain[len(bc.Chain)-1].Height
	return blockHeight
}
