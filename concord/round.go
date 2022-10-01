package concord

import (
	"bytes"
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/p2p"

	"github.com/Wondertan/iotwal/consensus/types"
)

type Round interface {
	// Propose takes a proposal block and gossipes it through the network.
	Propose(context.Context, Data)
	Vote(context.Context, *BlockID)
	PreCommit()

	addVote(*Vote, peer.ID) bool
}

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
	// TODO: move struct to iotwal
	voteSet       *types.HeightVoteSet
	proposedBlock *BlockID
}

func newRound(concordId string, lastValidRound int32, topic *pubsub.Topic, info *valInfo) Round {
	return &round{lastValidRound: lastValidRound, topic: topic, valInfo: info}
}

func (r *round) Propose(ctx context.Context, data Data) {
	if r.isProposer() {
		if err := r.propose(ctx, data); err != nil {
			panic(err)
		}
	}
}

func (r *round) propose(ctx context.Context, data Data) error {
	prop := NewProposal(0, r.lastValidRound+1, r.lastValidRound, BlockID{Hash: data.Hash()})
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

func (r *round) Vote(ctx context.Context, b *BlockID) {
	r.proposedBlock = b
	vote := NewVote(r.lastValidRound+1, b)
	proto := vote.ToProto()
	err := r.valInfo.valSelf.SignVote(r.concordId, proto)
	if err != nil {
		panic(err)
	}

	vote.Signature = proto.Signature
	if err = r.publish(ctx, &VoteMessage{Vote: vote}); err != nil {
		log.Errorw("during publishing", "err", err)
	}
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

func (r *round) PreCommit() {
}

func (r *round) addVote(v *Vote, from peer.ID) bool {
	added, err := r.voteSet.AddVote(nil, p2p.ID(from))
	if err != nil {
		panic(err)
	}
	return added
}

func (r *round) isProposer() bool {
	return bytes.Equal(r.valInfo.valSet.GetProposer().Address, r.valInfo.valSelfPK.Address())
}
