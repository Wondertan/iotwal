package concord

import (
	"bytes"
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/Wondertan/iotwal/concord/pb"
)

type propInfo struct {
	set       *ProposerSet
	self      PrivProposer
	selfIndex int32
	selfPK    crypto.PubKey
}

type round struct {
	concordId string
	topic     *pubsub.Topic
	valInfo   *propInfo

	round  int32
	propCh chan []byte
	votes  map[pb.SignedMsgType]*VoteSet
}

func newRound(r int, concordId string, topic *pubsub.Topic, info *propInfo) *round {
	return &round{
		concordId: concordId,
		topic:     topic,
		valInfo:   info,
		round:     int32(r),
		propCh:    make(chan []byte, 1),
		votes: map[pb.SignedMsgType]*VoteSet{
			pb.PrevoteType:   NewVoteSet(concordId, pb.PrevoteType, info.set),
			pb.PrecommitType: NewVoteSet(concordId, pb.PrecommitType, info.set),
		},
	}
}

// Propose takes a proposal block and gossipes it through the network.
// TODO: Timeouts
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

	return r.publish(ctx, pprop)
}

func (r *round) isProposer() bool {
	return bytes.Equal(r.valInfo.set.GetProposer().Address, r.valInfo.selfPK.Address())
}

func (r *round) rcvProposal(ctx context.Context, prop *Proposal) error {
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

// TODO: Timeouts
func (r *round) Vote(ctx context.Context, hash tmbytes.HexBytes, voteType pb.SignedMsgType) (*VoteSet, error) {
	err := r.vote(ctx, hash, voteType)
	if err != nil {
		return nil, err
	}

	select {
	case <-r.votes[voteType].Done():
		return r.votes[voteType], nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *round) vote(ctx context.Context, hash tmbytes.HexBytes, msgType pb.SignedMsgType) error {
	vote := NewVote(msgType, r.round, r.valInfo.selfIndex, r.valInfo.selfPK.Address(), &DataHash{Hash: hash})
	proto := vote.ToProto()
	err := r.valInfo.self.SignVote(r.concordId, proto)
	if err != nil {
		return err
	}

	return r.publish(ctx, proto)
}

func (r *round) rcvVote(_ context.Context, v *Vote) error {
	// adds the vote and does all the necessary verifications
	_, err := r.votes[v.Type].AddVote(v)
	return err
}

// rework publish to receive pb.Message
func (r *round) publish(ctx context.Context, message proto.Message) error {
	bin, err := encodeMsg(message)
	if err != nil {
		return err
	}
	return r.topic.Publish(ctx, bin)
}

var (
	ErrProposalSignature          = errors.New("invalid proposal signature")
	ErrProposalRound              = errors.New("invalid proposal round")
	ErrAddingVote                 = errors.New("adding vote")
	ErrSignatureFoundInPastBlocks = errors.New("found signature from the same key")
)
