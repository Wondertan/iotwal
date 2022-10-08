package concord

import (
	"encoding"

	logging "github.com/ipfs/go-log/v2"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

var log = logging.Logger("conciliator")

type Data interface {
	Hash() tmbytes.HexBytes
	encoding.BinaryMarshaler
}

type Consensus interface {
	AgreeOn(Data) (Data, error)
}