package concord

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Wondertan/iotwal/concord/pb"
	"github.com/tendermint/tendermint/libs/bits"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

func TestMsgToProto(t *testing.T) {
	bi := BlockID{
		Hash:          tmrand.Bytes(32),
	}
	pbBi := bi.ToProto()
	bits := bits.NewBitArray(1)
	pbBits := bits.ToProto()

	proposal := Proposal{
		Type:      pb.ProposalType,
		Height:    1,
		Round:     1,
		POLRound:  1,
		BlockID:   bi,
		Timestamp: time.Now(),
		Signature: tmrand.Bytes(20),
	}
	pbProposal := proposal.ToProto()

	pv := NewMockPV()
	pk, err := pv.GetPubKey()
	require.NoError(t, err)
	val := NewValidator(pk, 100)

	vote, err := MakeVote(
		1, BlockID{}, &ProposerSet{Proposer: val, Proposers: []*Proposer{val}},
		pv, "chainID", time.Now())
	require.NoError(t, err)
	pbVote := vote.ToProto()

	testsCases := []struct {
		testName string
		msg      Message
		want     *pb.Message
		wantErr  bool
	}{
		{"successful NewRoundStepMessage", &NewRoundStepMessage{
			Height:                2,
			Round:                 1,
			Step:                  1,
			SecondsSinceStartTime: 1,
			LastCommitRound:       2,
		}, &pb.Message{
			Sum: &pb.Message_NewRoundStep{
				NewRoundStep: &pb.NewRoundStep{
					Height:                2,
					Round:                 1,
					Step:                  1,
					SecondsSinceStartTime: 1,
					LastCommitRound:       2,
				},
			},
		}, false},

		{"successful NewValidBlockMessage", &NewValidBlockMessage{
			Height:             1,
			Round:              1,
			// BlockPartSetHeader: psh,
			// BlockParts:         bits,
			IsCommit:           false,
		}, &pb.Message{
			Sum: &pb.Message_NewValidBlock{
				NewValidBlock: &pb.NewValidBlock{
					Height:             1,
					Round:              1,
					// BlockPartSetHeader: pbPsh,
					// BlockParts:         pbBits,
					IsCommit:           false,
				},
			},
		}, false},
		{"successful ProposalPOLMessage", &ProposalPOLMessage{
			Height:           1,
			ProposalPOLRound: 1,
			ProposalPOL:      bits,
		}, &pb.Message{
			Sum: &pb.Message_ProposalPol{
				ProposalPol: &pb.ProposalPOL{
					Height:           1,
					ProposalPolRound: 1,
					ProposalPol:      *pbBits,
				},
			}}, false},
		{"successful ProposalMessage", &ProposalMessage{
			Proposal: &proposal,
		}, &pb.Message{
			Sum: &pb.Message_Proposal{
				Proposal: pbProposal,
			},
		}, false},
		{"successful VoteMessage", &VoteMessage{
			Vote: vote,
		}, &pb.Message{
			Sum: &pb.Message_Vote{
				Vote: pbVote,
			},
		}, false},
		{"successful VoteSetMaj23", &VoteSetMaj23Message{
			Height:  1,
			Round:   1,
			Type:    1,
			BlockID: bi,
		}, &pb.Message{
			Sum: &pb.Message_VoteSetMaj23{
				VoteSetMaj23: &pb.VoteSetMaj23{
					Height:  1,
					Round:   1,
					Type:    1,
					BlockID: pbBi,
				},
			},
		}, false},
		{"successful VoteSetBits", &VoteSetBitsMessage{
			Height:  1,
			Round:   1,
			Type:    1,
			BlockID: bi,
			Votes:   bits,
		}, &pb.Message{
			Sum: &pb.Message_VoteSetBits{
				VoteSetBits: &pb.VoteSetBits{
					Height:  1,
					Round:   1,
					Type:    1,
					BlockID: pbBi,
					Votes:   *pbBits,
				},
			},
		}, false},
		{"failure", nil, &pb.Message{}, true},
	}
	for _, tt := range testsCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			pb, err := MsgToProto(tt.msg)
			if tt.wantErr == true {
				assert.Equal(t, err != nil, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.want, pb, tt.testName)

			msg, err := MsgFromProto(pb)

			if !tt.wantErr {
				require.NoError(t, err)
				bcm := assert.Equal(t, tt.msg, msg, tt.testName)
				assert.True(t, bcm, tt.testName)
			} else {
				require.Error(t, err, tt.testName)
			}
		})
	}
}
