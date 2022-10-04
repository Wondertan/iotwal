package concord

import (
	"bytes"
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type executor interface {
	// execute creates and publishes prevote and precommit votes
	execute(context.Context, tmproto.SignedMsgType)
}

type Round interface {
	executor
	// Propose takes a proposal block and gossipes it through the network.
	Propose(context.Context, Data)

	addProposedBlock(*BlockID)
	proposedBlock() *BlockID
}

// temp solution. should be moved to concord
type valInfo struct {
	valSet    *ValidatorSet
	valSelf   PrivValidator
	valSelfPK crypto.PubKey
}

func newValinfo(set *ValidatorSet, valSelf PrivValidator, key crypto.PubKey) *valInfo {
	return &valInfo{set, valSelf, key}
}

type round struct {
	concordId      string
	lastValidRound int32
	topic          *pubsub.Topic
	valInfo        *valInfo
	proposalBlock  *BlockID
}

func newRound(concordId string, lastValidRound int32, topic *pubsub.Topic, info *valInfo) Round {
	return &round{concordId: concordId, lastValidRound: lastValidRound, topic: topic, valInfo: info}
}

func (r *round) Propose(ctx context.Context, data Data) {
	if r.isProposer() {
		if err := r.propose(ctx, data); err != nil {
			log.Errorw("during publishing", "err", err)
		}
	}
}

func (r *round) propose(ctx context.Context, data Data) error {
	r.proposalBlock = &BlockID{Hash: data.Hash()}
	prop := NewProposal(0, r.lastValidRound+1, r.lastValidRound, *r.proposalBlock)
	pprop := prop.ToProto()

	err := r.valInfo.valSelf.SignProposal(r.concordId, pprop)
	if err != nil {
		log.Errorw("signing proposal",
			"round", r.lastValidRound+1,
			"err", err,
		)
		return err
	}
	prop.Signature = pprop.Signature
	return r.publish(ctx, &ProposalMessage{Proposal: prop})
}

func (r *round) isProposer() bool {
	return bytes.Equal(r.valInfo.valSet.GetProposer().Address, r.valInfo.valSelfPK.Address())
}

func (r *round) execute(ctx context.Context, msgType tmproto.SignedMsgType) {
	vote := NewVote(msgType, r.lastValidRound+1, r.proposalBlock)
	proto := vote.ToProto()
	err := r.valInfo.valSelf.SignVote(r.concordId, proto)
	if err != nil {
		panic(err)
	}

	vote.Signature = proto.Signature
	if err = r.publish(ctx, &VoteMessage{Vote: vote}); err != nil {
		log.Errorw("during publishing", "err", err, "msgType", msgType)
	}
}

func (r *round) addProposedBlock(b *BlockID) {
	if r.isProposer() {
		// ensure that nobody changed choosen block
		if bytes.Equal(b.Hash, r.proposalBlock.Hash) {
			panic("invalid proposed block")
		}
		return
	}
	r.proposalBlock = b
}

func (r *round) proposedBlock() *BlockID {
	return r.proposalBlock
}

func (r *round) publish(ctx context.Context, message Message) error {
	proto, err := MsgToProto(message)
	if err != nil {
		return err
	}

	bin, err := proto.Marshal()
	if err != nil {
		return err
	}

	return r.topic.Publish(ctx, bin)
}
