package concord

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/protoio"

	"github.com/Wondertan/iotwal/concord/pb"
)

func examplePrevote() *Vote {
	return exampleVote(byte(pb.PrevoteType))
}

func examplePrecommit() *Vote {
	return exampleVote(byte(pb.PrecommitType))
}

func exampleVote(t byte) *Vote {
	var stamp, err = time.Parse(RFC3339Nano, "2017-12-25T03:00:01.234Z")
	if err != nil {
		panic(err)
	}

	return &Vote{
		Type:      pb.SignedMsgType(t),
		Timestamp: stamp,
		DataHash: DataHash{
			Hash: tmhash.Sum([]byte("blockID_hash")),
		},
		ValidatorAddress: crypto.AddressHash([]byte("validator_address")),
		ValidatorIndex:   56789,
		ChainID:          "test_chain_ID",
	}
}

func TestVoteSignable(t *testing.T) {
	vote := examplePrecommit()
	v := vote.ToProto()
	signBytes := VoteSignBytes(v)
	expected, err := protoio.MarshalDelimited(v)
	require.NoError(t, err)

	require.Equal(t, expected, signBytes, "Got unexpected sign bytes for Vote.")
}

func TestVoteSignBytesTestVectors(t *testing.T) {

	tests := []struct {
		vote *Vote
		want []byte
	}{
		0: {
			&Vote{},
			// NOTE: Height and Round are skipped here. This case needs to be considered while parsing.
			[]byte{0xd, 0x2a, 0xb, 0x8, 0x80, 0x92, 0xb8, 0xc3, 0x98, 0xfe, 0xff, 0xff, 0xff, 0x1},
		},
		// with proper (fixed size) height and round (PreCommit):
		1: {
			&Vote{Type: pb.PrecommitType, ChainID: ""},
			[]byte{
				0x21,                                   // length
				0x8,                                    // (field_number << 3) | wire_type
				0x2,                                    // PrecommitType
				0x11,                                   // (field_number << 3) | wire_type
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // height
				0x19,                                   // (field_number << 3) | wire_type
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // round
				0x2a, // (field_number << 3) | wire_type
				// remaining fields (timestamp):
				0xb, 0x8, 0x80, 0x92, 0xb8, 0xc3, 0x98, 0xfe, 0xff, 0xff, 0xff, 0x1},
		},
		// with proper (fixed size) height and round (PreVote):
		2: {
			&Vote{Type: pb.PrevoteType, ChainID: ""},
			[]byte{
				0x21,                                   // length
				0x8,                                    // (field_number << 3) | wire_type
				0x1,                                    // PrevoteType
				0x11,                                   // (field_number << 3) | wire_type
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // height
				0x19,                                   // (field_number << 3) | wire_type
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // round
				0x2a, // (field_number << 3) | wire_type
				// remaining fields (timestamp):
				0xb, 0x8, 0x80, 0x92, 0xb8, 0xc3, 0x98, 0xfe, 0xff, 0xff, 0xff, 0x1},
		},
	}
	for i, tc := range tests {
		v := tc.vote.ToProto()
		got := VoteSignBytes(v)
		assert.Equal(t, len(tc.want), len(got), "test case #%v: got unexpected sign bytes length for Vote.", i)
		assert.Equal(t, tc.want, got, "test case #%v: got unexpected sign bytes for Vote.", i)
	}
}

func TestVoteVerifySignature(t *testing.T) {
	privVal := NewMockPV()
	pubkey, err := privVal.GetPubKey()
	require.NoError(t, err)

	vote := examplePrecommit()
	v := vote.ToProto()
	signBytes := VoteSignBytes(v)

	// sign it
	err = privVal.SignVote("test_chain_id", v)
	require.NoError(t, err)

	// verify the same vote
	valid := pubkey.VerifySignature(VoteSignBytes(v), v.Signature)
	require.True(t, valid)

	// serialize, deserialize and verify again....
	precommit := new(pb.Vote)
	bs, err := proto.Marshal(v)
	require.NoError(t, err)
	err = proto.Unmarshal(bs, precommit)
	require.NoError(t, err)

	// verify the transmitted vote
	newSignBytes := VoteSignBytes(precommit)
	require.Equal(t, string(signBytes), string(newSignBytes))
	valid = pubkey.VerifySignature(newSignBytes, precommit.Signature)
	require.True(t, valid)
}

func TestIsVoteTypeValid(t *testing.T) {
	tc := []struct {
		name string
		in   pb.SignedMsgType
		out  bool
	}{
		{"Prevote", pb.PrevoteType, true},
		{"Precommit", pb.PrecommitType, true},
		{"InvalidType", pb.SignedMsgType(0x3), false},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(st *testing.T) {
			if rs := IsVoteTypeValid(tt.in); rs != tt.out {
				t.Errorf("got unexpected Vote type. Expected:\n%v\nGot:\n%v", rs, tt.out)
			}
		})
	}
}

func TestVoteVerify(t *testing.T) {
	privVal := NewMockPV()
	pubkey, err := privVal.GetPubKey()
	require.NoError(t, err)

	vote := examplePrevote()
	vote.ValidatorAddress = pubkey.Address()

	err = vote.Verify(ed25519.GenPrivKey().PubKey())
	if assert.Error(t, err) {
		assert.Equal(t, ErrVoteInvalidValidatorAddress, err)
	}

	err = vote.Verify(pubkey)
	if assert.Error(t, err) {
		assert.Equal(t, ErrVoteInvalidSignature, err)
	}
}

func TestVoteString(t *testing.T) {
	str := examplePrecommit().String()
	expected := `Vote{56789:6AF1F4111082 12345/02/SIGNED_MSG_TYPE_PRECOMMIT(Precommit) 8B01023386C3 000000000000 @ 2017-12-25T03:00:01.234Z}` //nolint:lll //ignore line length for tests
	if str != expected {
		t.Errorf("got unexpected string for Vote. Expected:\n%v\nGot:\n%v", expected, str)
	}

	str2 := examplePrevote().String()
	expected = `Vote{56789:6AF1F4111082 12345/02/SIGNED_MSG_TYPE_PREVOTE(Prevote) 8B01023386C3 000000000000 @ 2017-12-25T03:00:01.234Z}` //nolint:lll //ignore line length for tests
	if str2 != expected {
		t.Errorf("got unexpected string for Vote. Expected:\n%v\nGot:\n%v", expected, str2)
	}
}

func TestVoteValidateBasic(t *testing.T) {
	privVal := NewMockPV()

	testCases := []struct {
		testName     string
		malleateVote func(*Vote)
		expectErr    bool
	}{
		{"Good Vote", func(v *Vote) {}, false},
		{"Invalid BlockID", func(v *Vote) {
			v.DataHash = DataHash{[]byte{1, 2, 3}}
		}, true},
		{"Invalid Address", func(v *Vote) { v.ValidatorAddress = make([]byte, 1) }, true},
		{"Invalid ValidatorIndex", func(v *Vote) { v.ValidatorIndex = -1 }, true},
		{"Invalid Signature", func(v *Vote) { v.Signature = nil }, true},
		{"Too big Signature", func(v *Vote) { v.Signature = make([]byte, MaxSignatureSize+1) }, true},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			vote := examplePrecommit()
			v := vote.ToProto()
			err := privVal.SignVote("test_chain_id", v)
			vote.Signature = v.Signature
			require.NoError(t, err)
			tc.malleateVote(vote)
			assert.Equal(t, tc.expectErr, vote.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestVoteProtobuf(t *testing.T) {
	privVal := NewMockPV()
	vote := examplePrecommit()
	v := vote.ToProto()
	err := privVal.SignVote("test_chain_id", v)
	vote.Signature = v.Signature
	require.NoError(t, err)

	testCases := []struct {
		msg     string
		v1      *Vote
		expPass bool
	}{
		{"success", vote, true},
		{"fail vote validate basic", &Vote{}, false},
		{"failure nil", nil, false},
	}
	for _, tc := range testCases {
		protoProposal := tc.v1.ToProto()

		v, err := VoteFromProto(protoProposal)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.v1, v, tc.msg)
		} else {
			require.Error(t, err)
		}
	}
}
