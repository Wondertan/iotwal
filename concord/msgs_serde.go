package concord

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"

	cpb "github.com/Wondertan/iotwal/concord/pb"
)

// MsgToProto takes a consensus message type and returns the proto defined consensus message
func MsgToProto(msg Message) (*cpb.Message, error) {
	if msg == nil {
		return nil, errors.New("consensus: message is nil")
	}
	var pb cpb.Message

	switch msg := msg.(type) {
	case *ProposalMessage:
		pbP := msg.Proposal.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_Proposal{
				Proposal: pbP,
			},
		}
	case *VoteMessage:
		vote := msg.Vote.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_Vote{
				Vote: vote,
			},
		}
	default:
		return nil, fmt.Errorf("consensus: message not recognized: %T", msg)
	}

	return &pb, nil
}

// MsgFromProto takes a consensus proto message and returns the native go type
func MsgFromProto(msg *cpb.Message) (Message, error) {
	if msg == nil {
		return nil, errors.New("consensus: nil message")
	}
	var pb Message

	switch msg := msg.Sum.(type) {
	case *cpb.Message_Proposal:
		pbP, err := ProposalFromProto(msg.Proposal)
		if err != nil {
			return nil, fmt.Errorf("proposal msg to proto error: %w", err)
		}

		pb = &ProposalMessage{
			Proposal: pbP,
		}
	case *cpb.Message_Vote:
		vote, err := VoteFromProto(msg.Vote)
		if err != nil {
			return nil, fmt.Errorf("vote msg to proto error: %w", err)
		}

		pb = &VoteMessage{
			Vote: vote,
		}
	default:
		return nil, fmt.Errorf("consensus: message not recognized: %T", msg)
	}

	if err := pb.ValidateBasic(); err != nil {
		return nil, err
	}

	return pb, nil
}

// MustEncode takes the reactors msg, makes it proto and marshals it
// this mimics `MustMarshalBinaryBare` in that is panics on error
func MustEncode(msg Message) []byte {
	pb, err := MsgToProto(msg)
	if err != nil {
		panic(err)
	}
	enc, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return enc
}
