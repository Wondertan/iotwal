package concord

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cosmos/gogoproto/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/Wondertan/iotwal/concord/pb"
)

type ProposerStore interface {
	Get(context.Context, string) (*ProposerSet, error)
}

type Validator func(context.Context, []byte) (tmbytes.HexBytes, error)

type conciliator struct {
	pubsub *pubsub.PubSub

	propStore ProposerStore
	propSelf  PrivProposer
}

func NewConciliator(p *pubsub.PubSub, self PrivProposer) *conciliator {
	return &conciliator{pubsub: p, propSelf: self}
}

// TODO:
//   - Cache for proposals and votes received from the PS. For rounds that hasn't been started on
//     local instance yet
//   - Garbage collect the the cache every like minute
//   - Limit cache to receive no more than N messages from a concord peer
type concord struct {
	id    string
	topic *pubsub.Topic

	roundMu sync.Mutex
	round   *round

	validate  Validator
	propStore ProposerStore
	self      PrivProposer
	selfPK    crypto.PubKey

	// allows only one AgreeOn call
	agreeLk sync.Mutex

	subs *pubsub.Subscription
}

func (c *conciliator) NewConcord(id string, pv Validator) (*concord, error) {
	// TODO: There should be at least one subscription
	tpc, err := c.pubsub.Join(id)
	if err != nil {
		return nil, err
	}

	pk, err := c.propSelf.GetPubKey()
	if err != nil {
		return nil, err
	}

	cord := &concord{
		id:        id,
		topic:     tpc,
		validate:  pv,
		propStore: c.propStore,
		self:      c.propSelf,
		selfPK:    pk,
	}
	cord.subs, err = tpc.Subscribe()
	if err != nil {
		return nil, err
	}
	return cord, c.pubsub.RegisterTopicValidator(id, cord.incoming)
}

// TODO:
//   - Consider passing ProposerSet as a param
//   - Implement vote for nil and timeouts and multi round voting
//   - Introduce another 'session' entity identified by prop hash
//   - Enables multiple independent agreements
//   - Fixes potential catching up issues for layers above
//   - Handle case where our thread is blocked on validation, but majority already locked on the block.
//   - Possible during catching up
func (c *concord) AgreeOn(ctx context.Context, propSet *ProposerSet, prop []byte) ([]byte, *Commit, error) {
	if propSet == nil {
		return nil, nil, errors.New("concord: empty proposer set")
	}
	c.agreeLk.Lock()
	defer c.agreeLk.Unlock()

	c.roundMu.Lock()
	index := -1
	for i, set := range propSet.Proposers {
		if bytes.Equal(set.PubKey.Bytes(), c.selfPK.Bytes()) {
			index = i
		}
	}
	if index < 0 {
		panic("could not find validator index")
	}
	c.round = newRound(0, c.id, c.topic, &propInfo{propSet, c.self, int32(index), c.selfPK})
	c.roundMu.Unlock()

	prop, err := c.round.Propose(ctx, prop)
	if err != nil {
		return nil, nil, err
	}

	hash, err := c.validate(ctx, prop)
	if err != nil {
		return nil, nil, err
	}

	_, err = c.round.Vote(ctx, hash, pb.PrevoteType)
	if err != nil {
		return nil, nil, err
	}
	// TODO: Check for double signing/equivocation
	// TODO: Do we need to wait for all the votes or can we send PreCommits right after?
	votes, err := c.round.Vote(ctx, hash, pb.PrecommitType)
	if err != nil {
		return nil, nil, err
	}

	return prop, votes.MakeCommit(), nil
}

func (c *concord) incoming(ctx context.Context, _ peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
	err := c.handle(ctx, pmsg)
	if err != nil {
		return pubsub.ValidationReject
	}

	return pubsub.ValidationAccept
}

func (c *concord) handle(ctx context.Context, pmsg *pubsub.Message) error {
	tmsg := &pb.Message{}
	err := proto.Unmarshal(pmsg.Data, tmsg)
	if err != nil {
		return err
	}

	msg, err := MsgFromProto(tmsg)
	if err != nil {
		return err
	}

	err = msg.ValidateBasic()
	if err != nil {
		return err
	}

	c.roundMu.Lock()
	round := c.round
	c.roundMu.Unlock()

	switch tmsg.Sum.(type) {
	case *pb.Message_Proposal:
		prop, err := ProposalFromProto(tmsg.GetProposal())
		if err != nil {
			return err
		}
		return round.rcvProposal(ctx, prop)
	case *pb.Message_Vote:
		vote, err := VoteFromProto(tmsg.GetVote())
		if err != nil {
			return err
		}
		return round.rcvVote(ctx, vote)
	default:
		return errors.New("concord: invalid message received")
	}
}

// encodeMsg encodes a Protobuf message
func encodeMsg(m proto.Message) ([]byte, error) {
	msg := &pb.Message{}
	switch t := m.(type) {
	case *pb.Proposal:
		msg.Sum = &pb.Message_Proposal{Proposal: t}
	case *pb.Vote:
		msg.Sum = &pb.Message_Vote{Vote: t}
	default:
		return nil, fmt.Errorf("unknown message type %T", m)
	}

	bz, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal %T: %w", msg, err)
	}

	return bz, nil
}
