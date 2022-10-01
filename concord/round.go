package concord

import (
	"context"

	"github.com/gogo/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Round interface {
	// Propose takes a proposal block and gossipes it through the network.
	Propose(context.Context, *Proposal)
	Vote(context.Context, *Vote)
	PreCommit()
}

type round struct {
	lastValidRound int64
	roundStepType  RoundStepType
	topic          *pubsub.Topic
}

func newRound(lastValidRound int64, topic *pubsub.Topic) Round {
	return &round{lastValidRound: lastValidRound, roundStepType: RoundStepNewRound, topic: topic}
}

func (r *round) Propose(ctx context.Context, p *Proposal) {
	b, _ := proto.Marshal(p.ToProto())
	r.roundStepType = RoundStepPropose
	_ = r.topic.Publish(ctx, b)
}

func (r *round) Vote(ctx context.Context, v *Vote) {
	if r.roundStepType != RoundStepPropose {
		panic("invalid round")
	}
	b, _ := proto.Marshal(v.ToProto())
	r.roundStepType = RoundStepPrevote
	_ = r.topic.Publish(ctx, b)
}

func (r *round) PreCommit() {
}
