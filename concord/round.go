package concord

import (
	"bytes"
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/Wondertan/iotwal/concord/pb"
)

type propInfo struct {
	set    *ProposerSet
	self   PrivValidator
	selfPK crypto.PubKey
}

type round struct {
	concordId      string
	topic          *pubsub.Topic
	valInfo        *propInfo

	round int32
	votes          *HeightVoteSet

	propCh chan []byte
	voteCh chan *Vote
	commCh chan *Vote

}

func newRound(concordId string, topic *pubsub.Topic, info *propInfo) *round {
	return &round{
		concordId: concordId,
		topic:     topic,
		valInfo:   info,
		propCh:    make(chan []byte),
	}
}

// Propose takes a proposal block and gossipes it through the network.
func (r *round) Propose(ctx context.Context, data []byte) ([]byte, error) {
	if r.isProposer() {
		err := r.propose(ctx, data)
		if err != nil {
			return nil, err
		}
	}

	select {
	case prop := <-r.propCh:
		return prop, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *round) propose(ctx context.Context, data []byte) error {
	prop := NewProposal(r.round, r.round, data)
	pprop := prop.ToProto()

	err := r.valInfo.self.SignProposal(r.concordId, pprop)
	if err != nil {
		return err
	}
	prop.Signature = pprop.Signature

	return r.publish(ctx, &ProposalMessage{Proposal: prop})
}

func (r *round) isProposer() bool {
	return bytes.Equal(r.valInfo.set.GetProposer().Address, r.valInfo.selfPK.Address())
}

func (r *round) rcvProposal(ctx context.Context, prop *Proposal) error {
	if prop.Round != prop.Round {
		return ErrProposalRound
	}

	// TODO Verify POLRound
	proper := r.valInfo.set.GetProposer().PubKey
	if !proper.VerifySignature(ProposalSignBytes(r.concordId, prop.ToProto()), prop.Signature) {
		return ErrProposalSignature
	}

	select {
	case r.propCh <- prop.Data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}


func (r *round) Vote(ctx context.Context, hash tmbytes.HexBytes) error {
	err := r.execute(ctx, hash, pb.PrevoteType)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *round) PreCommit(ctx context.Context, hash tmbytes.HexBytes) error {
	err := r.execute(ctx, hash, pb.PrecommitType)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *round) execute(ctx context.Context, hash tmbytes.HexBytes, msgType pb.SignedMsgType) error {
	vote := NewVote(msgType, r.round, &BlockID{Hash: hash})
	proto := vote.ToProto()
	err := r.valInfo.self.SignVote(r.concordId, proto)
	if err != nil {
		return err
	}
	vote.Signature = proto.Signature

	return r.publish(ctx, &VoteMessage{Vote: vote})
}

func (r *round) addVote(vote *Vote, p peer.ID) (bool, error) {
	if vote.Round == r.round && vote.Type == pb.PrecommitType {
		log.Debug("received vote for the previous round")
		return false, nil // TODO: make be return a specific error???
	}

	return r.votes.AddVote(vote, p)
}

func (r *round) rcvVote(_ context.Context, v *Vote, from peer.ID) error {
	_, err := r.votes.AddVote(v, from)
	if err != nil {
		return err
	}

	return nil
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

var (
	ErrProposalSignature = errors.New("invalid proposal signature")
	ErrProposalRound     = errors.New("invalid proposal round")
	ErrAddingVote               = errors.New("adding vote")
	ErrSignatureFoundInPastBlocks = errors.New("found signature from the same key")
)
