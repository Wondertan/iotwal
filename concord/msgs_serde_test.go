package concord

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmrand "github.com/tendermint/tendermint/libs/rand"

	"github.com/Wondertan/iotwal/concord/pb"
)

func TestMsgToProto(t *testing.T) {
	proposal := Proposal{
		Type:      pb.ProposalType,
		Round:     1,
		POLRound:  1,
		Timestamp: time.Now(),
		Signature: tmrand.Bytes(20),
	}
	pbProposal := proposal.ToProto()

	pv := NewMockPV()
	pk, err := pv.GetPubKey()
	require.NoError(t, err)
	val := NewValidator(pk, 100)

	vote, err := MakeVote(
		1, DataHash{}, &ProposerSet{Proposer: val, Proposers: []*Proposer{val}},
		pv, "chainID", time.Now())
	require.NoError(t, err)
	pbVote := vote.ToProto()

	testsCases := []struct {
		testName string
		msg      Message
		want     *pb.Message
		wantErr  bool
	}{
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
