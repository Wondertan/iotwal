package concord

import (
	"bytes"
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"

	"github.com/Wondertan/iotwal/concord/pb"
)

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

type round struct {
	concordId      string
	topic          *pubsub.Topic
	valInfo        *valInfo
	proposalBlock  *BlockID
	votes          *HeightVoteSet

	round int32

	waitProp chan []byte // marshalled Data
}

func newRound(concordId string, topic *pubsub.Topic, info *valInfo) *round {
	return &round{
		concordId: concordId,
		topic: topic,
		valInfo: info,
		waitProp: make(chan []byte),
	}
}

// Propose takes a proposal block and gossipes it through the network.
func (r *round) Propose(ctx context.Context, data Data) ([]byte, error) {
	if r.isProposer() {
		err := r.propose(ctx, data)
		if err != nil {
			return nil, err
		}
	}

	select {
	case prop := <-r.waitProp:
		return prop, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *round) propose(ctx context.Context, data Data) error {
	bin, err := data.MarshalBinary()
	if err != nil {
		return err
	}

	prop := NewProposal(0, r.round, r.round, BlockID{}, bin)
	pprop := prop.ToProto()

	err = r.valInfo.valSelf.SignProposal(r.concordId, pprop)
	if err != nil {
		return err
	}
	prop.Signature = pprop.Signature
	return r.publish(ctx, &ProposalMessage{Proposal: prop})
}

func (r *round) isProposer() bool {
	return bytes.Equal(r.valInfo.valSet.GetProposer().Address, r.valInfo.valSelfPK.Address())
}

func (r *round) execute(ctx context.Context, msgType pb.SignedMsgType) {
	vote := NewVote(msgType, r.round, r.proposalBlock)
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
	if vote.Round == r.round && vote.Type == pb.PrecommitType {
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

func (r *round) rcvVote(_ context.Context, v *Vote, from peer.ID) error {
	_, err := r.votes.AddVote(v, from)
	if err != nil {
		return err
	}

	return nil
}

func (r *round) rcvProposal(ctx context.Context, prop *Proposal) error {
	if prop.Round != prop.Round {
		return ErrProposalRound
	}

	// TODO Verify POLRound
	proper := r.valInfo.valSet.GetProposer().PubKey
	if !proper.VerifySignature(ProposalSignBytes(r.concordId, prop.ToProto()), prop.Signature) {
		return ErrProposalSignature
	}

	select {
	case r.waitProp <- prop.Data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}


var (
	ErrProposalSignature = errors.New("invalid proposal signature")
	ErrProposalRound     = errors.New("invalid proposal round")
	ErrAddingVote               = errors.New("adding vote")
	ErrSignatureFoundInPastBlocks = errors.New("found signature from the same key")
)
