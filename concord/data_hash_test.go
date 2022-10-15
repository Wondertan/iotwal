package concord

import (
	"math/rand"

	"github.com/tendermint/tendermint/crypto/tmhash"
)

func makeDataHashRandom() DataHash {
	var (
		blockHash   = make([]byte, tmhash.Size)
		partSetHash = make([]byte, tmhash.Size)
	)
	rand.Read(blockHash)   //nolint: errcheck // ignore errcheck for read
	rand.Read(partSetHash) //nolint: errcheck // ignore errcheck for read
	return DataHash{blockHash}
}

func makeDataHash(hash []byte, partSetHash []byte) DataHash {
	var (
		h   = make([]byte, tmhash.Size)
		psH = make([]byte, tmhash.Size)
	)
	copy(h, hash)
	copy(psH, partSetHash)
	return DataHash{
		Hash: h,
	}
}
