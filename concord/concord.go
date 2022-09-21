package concord

import (
	cstypes "github.com/Wondertan/iotwal/consensus/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

var log = logging.Logger("conciliator")

type Data interface {
	Hash() tmbytes.HexBytes
	Validate(Data) (bool, error)
}

type Consensus interface {
	AgreeOn(Data) (Data, error)
}

type RoundState struct {
	Height    int64
	Round     int32
	Step      RoundStepType `json:"step"`

	Validators *ValidatorSet
}

type consensus struct {
	privValidator PrivValidator
	privValidatorPubKey crypto.PubKey
	valset *ValidatorSet

	cstypes.RoundState
}

func (c *consensus) AgreeOn(data Data) (Data, error) {

	return data, nil
}
