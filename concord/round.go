package concord

import (
	"bytes"
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"

	"github.com/Wondertan/iotwal/concord/pb"
)

type executor interface {
	// execute creates and publishes prevote and precommit votes
	execute(context.Context, pb.SignedMsgType)
}

type Round interface {
	// Propose takes a proposal block and gossips it through the network.
	Propose(context.Context, Data)
	execute(context.Context, pb.SignedMsgType)

	addVote(*Vote, peer.ID) (bool, error)

	preCommits() *VoteSet
	preVotes() *VoteSet

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
	lastValidRound int64
	topic          *pubsub.Topic
	valInfo        *valInfo
	proposalBlock  *BlockID
	votes          *HeightVoteSet
}

func newRound(concordId string, lastValidRound int64, topic *pubsub.Topic, info *valInfo) Round {
	return &round{
		concordId:      concordId,
		lastValidRound: lastValidRound,
		topic:          topic,
		valInfo:        info,
		votes:          NewHeightVoteSet(concordId, lastValidRound+1, info.valSet),
	}
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
	prop := NewProposal(0, int32(r.lastValidRound+1), int32(r.lastValidRound), *r.proposalBlock)
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

func (r *round) execute(ctx context.Context, msgType pb.SignedMsgType) {
	vote := NewVote(msgType, int32(r.lastValidRound+1), r.proposalBlock)
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
		// ensure that nobody changed chosen block
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

func (r *round) addVote(vote *Vote, p peer.ID) (bool, error) {
	if vote.Height == r.lastValidRound && vote.Type == pb.PrecommitType {
		log.Debug("received vote for the previous round")
		return false, nil // TODO: make be return a specific error???
	}

	return r.votes.AddVote(vote, p)
}

func (r *round) preVotes() *VoteSet {
	return r.votes.Prevotes(0)
}

func (r *round) preCommits() *VoteSet {
	return r.votes.Precommits(0)
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
