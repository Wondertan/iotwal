package concord

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/tendermint/tendermint/libs/bits"
	tmmath "github.com/tendermint/tendermint/libs/math"

	cpb "github.com/Wondertan/iotwal/concord/pb"
)

// MsgToProto takes a consensus message type and returns the proto defined consensus message
func MsgToProto(msg Message) (*cpb.Message, error) {
	if msg == nil {
		return nil, errors.New("consensus: message is nil")
	}
	var pb cpb.Message

	switch msg := msg.(type) {
	case *NewRoundStepMessage:
		pb = cpb.Message{
			Sum: &cpb.Message_NewRoundStep{
				NewRoundStep: &cpb.NewRoundStep{
					Height:                msg.Height,
					Round:                 msg.Round,
					Step:                  uint32(msg.Step),
					SecondsSinceStartTime: msg.SecondsSinceStartTime,
					LastCommitRound:       msg.LastCommitRound,
				},
			},
		}
	case *NewValidBlockMessage:
		// pbPartSetHeader := msg.BlockPartSetHeader.ToProto()
		// pbBits := msg.BlockParts.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_NewValidBlock{
				NewValidBlock: &cpb.NewValidBlock{
					Height:   msg.Height,
					Round:    msg.Round,
					IsCommit: msg.IsCommit,
				},
			},
		}
	case *ProposalMessage:
		pbP := msg.Proposal.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_Proposal{
				Proposal: pbP,
			},
		}
	case *ProposalPOLMessage:
		pbBits := msg.ProposalPOL.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_ProposalPol{
				ProposalPol: &cpb.ProposalPOL{
					Height:           msg.Height,
					ProposalPolRound: msg.ProposalPOLRound,
					ProposalPol:      *pbBits,
				},
			},
		}
	case *VoteMessage:
		vote := msg.Vote.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_Vote{
				Vote: vote,
			},
		}
	case *HasVoteMessage:
		pb = cpb.Message{
			Sum: &cpb.Message_HasVote{
				HasVote: &cpb.HasVote{
					Height: msg.Height,
					Round:  msg.Round,
					Type:   msg.Type,
					Index:  msg.Index,
				},
			},
		}
	case *VoteSetMaj23Message:
		bi := msg.DataHash.ToProto()
		pb = cpb.Message{
			Sum: &cpb.Message_VoteSetMaj23{
				VoteSetMaj23: &cpb.VoteSetMaj23{
					Height:   msg.Height,
					Round:    msg.Round,
					Type:     msg.Type,
					DataHash: *bi,
				},
			},
		}
	case *VoteSetBitsMessage:
		bi := msg.DataHash.ToProto()
		bits := msg.Votes.ToProto()

		vsb := &cpb.Message_VoteSetBits{
			VoteSetBits: &cpb.VoteSetBits{
				Height:   msg.Height,
				Round:    msg.Round,
				Type:     msg.Type,
				DataHash: *bi,
			},
		}

		if bits != nil {
			vsb.VoteSetBits.Votes = *bits
		}

		pb = cpb.Message{
			Sum: vsb,
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
	case *cpb.Message_NewRoundStep:
		rs, err := tmmath.SafeConvertUint8(int64(msg.NewRoundStep.Step))
		// deny message based on possible overflow
		if err != nil {
			return nil, fmt.Errorf("denying message due to possible overflow: %w", err)
		}
		pb = &NewRoundStepMessage{
			Height:                msg.NewRoundStep.Height,
			Round:                 msg.NewRoundStep.Round,
			Step:                  RoundStepType(rs),
			SecondsSinceStartTime: msg.NewRoundStep.SecondsSinceStartTime,
			LastCommitRound:       msg.NewRoundStep.LastCommitRound,
		}
	case *cpb.Message_NewValidBlock:
		pbBits := new(bits.BitArray)
		pbBits.FromProto(msg.NewValidBlock.BlockParts)

		pb = &NewValidBlockMessage{
			Height:   msg.NewValidBlock.Height,
			Round:    msg.NewValidBlock.Round,
			IsCommit: msg.NewValidBlock.IsCommit,
		}
	case *cpb.Message_Proposal:
		pbP, err := ProposalFromProto(msg.Proposal)
		if err != nil {
			return nil, fmt.Errorf("proposal msg to proto error: %w", err)
		}

		pb = &ProposalMessage{
			Proposal: pbP,
		}
	case *cpb.Message_ProposalPol:
		pbBits := new(bits.BitArray)
		pbBits.FromProto(&msg.ProposalPol.ProposalPol)
		pb = &ProposalPOLMessage{
			Height:           msg.ProposalPol.Height,
			ProposalPOLRound: msg.ProposalPol.ProposalPolRound,
			ProposalPOL:      pbBits,
		}
	case *cpb.Message_Vote:
		vote, err := VoteFromProto(msg.Vote)
		if err != nil {
			return nil, fmt.Errorf("vote msg to proto error: %w", err)
		}

		pb = &VoteMessage{
			Vote: vote,
		}
	case *cpb.Message_HasVote:
		pb = &HasVoteMessage{
			Height: msg.HasVote.Height,
			Round:  msg.HasVote.Round,
			Type:   msg.HasVote.Type,
			Index:  msg.HasVote.Index,
		}
	case *cpb.Message_VoteSetMaj23:
		bi, err := DataHashFromProto(&msg.VoteSetMaj23.DataHash)
		if err != nil {
			return nil, fmt.Errorf("voteSetMaj23 msg to proto error: %w", err)
		}
		pb = &VoteSetMaj23Message{
			Height:   msg.VoteSetMaj23.Height,
			Round:    msg.VoteSetMaj23.Round,
			Type:     msg.VoteSetMaj23.Type,
			DataHash: *bi,
		}
	case *cpb.Message_VoteSetBits:
		bi, err := DataHashFromProto(&msg.VoteSetBits.DataHash)
		if err != nil {
			return nil, fmt.Errorf("voteSetBits msg to proto error: %w", err)
		}
		bits := new(bits.BitArray)
		bits.FromProto(&msg.VoteSetBits.Votes)

		pb = &VoteSetBitsMessage{
			Height:   msg.VoteSetBits.Height,
			Round:    msg.VoteSetBits.Round,
			Type:     msg.VoteSetBits.Type,
			DataHash: *bi,
			Votes:    bits,
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
