package concord

import (
	"bytes"
	"context"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
)

type conciliator struct {
	pubsub *pubsub.PubSub
}

type concord struct {
	concordId string
	topic     *pubsub.Topic

	valSet    *ValidatorSet
	valSelf   PrivValidator
	valSelfPK crypto.PubKey

	stateMu sync.Mutex
	state   struct {
		Height int64
		Round  int32
		Step   RoundStepType
		// Last known round with POL for non-nil valid block.
		ValidRound int32
	}
}

func (c *concord) AgreeOn(ctx context.Context, data Data) (Data, error) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.isProposer() {
		if err := c.propose(ctx, data); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (c *concord) propose(ctx context.Context, data Data) error {
	height, round := c.state.Height, c.state.Round

	prop := NewProposal(height, round, c.state.ValidRound, BlockID{Hash: data.Hash()})
	pprop := prop.ToProto()

	err := c.valSelf.SignProposal(c.concordId, pprop)
	if err != nil {
		log.Errorw("signing proposal",
			"height", height,
			"round", round,
			"err", err,
		)
		return err
	}
	prop.Signature = pprop.Signature

	return nil
}


func (c *concord) isProposer() bool {
	return bytes.Equal(c.valSet.GetProposer().Address, c.valSelfPK.Address())
}
