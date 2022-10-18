package concord

import (
	"errors"
	"fmt"
	"time"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/protoio"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/Wondertan/iotwal/concord/pb"
)

// Proposal defines a block proposal for the consensus.
// It refers to the block by BlockID field.
// It must be signed by the correct proposer for the given Height/Round
// to be considered valid. It may depend on votes from a previous round,
// a so-called Proof-of-Lock (POL) round, as noted in the POLRound.
// If POLRound >= 0, then BlockID corresponds to the block that is locked in POLRound.
type Proposal struct {
	Type      pb.SignedMsgType
	Round     int32     `json:"round"`     // there can not be greater than 2_147_483_647 rounds
	POLRound  int32     `json:"pol_round"` // -1 if null.
	Timestamp time.Time `json:"timestamp"`
	Signature []byte    `json:"signature"`
	Data      []byte    `json:"data"`
	ChainID   string
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(round int32, polRound int32, data []byte, chainID string) *Proposal {
	return &Proposal{
		Type:      pb.ProposalType,
		Round:     round,
		POLRound:  polRound,
		Timestamp: tmtime.Now(),
		Data:      data,
		ChainID:   chainID,
	}
}

// ValidateBasic performs basic validation.
func (p *Proposal) ValidateBasic() error {
	if p.Type != pb.ProposalType {
		return errors.New("invalid Type")
	}
	if p.Round < 0 {
		return errors.New("negative Round")
	}
	if p.POLRound < -1 {
		return errors.New("negative POLRound (exception: -1)")
	}

	// NOTE: Timestamp validation is subtle and handled elsewhere.

	if len(p.Signature) == 0 {
		return errors.New("signature is missing")
	}

	if len(p.Signature) > MaxSignatureSize {
		return fmt.Errorf("signature is too big (max: %d)", MaxSignatureSize)
	}
	return nil
}

// String returns a string representation of the Proposal.
//
// 1. height
// 2. round
// 3. block ID
// 4. POL round
// 5. first 6 bytes of signature
// 6. timestamp
//
// See BlockID#String.
func (p *Proposal) String() string {
	return fmt.Sprintf("Proposal{%v (%v) %X @ %s}",
		p.Round,
		p.POLRound,
		tmbytes.Fingerprint(p.Signature),
		CanonicalTime(p.Timestamp))
}

// ProposalSignBytes returns the proto-encoding of the Proposal,
// for signing. Panics if the marshaling fails.
//
// The encoded Protobuf message is varint length-prefixed (using MarshalDelimited)
// for backwards-compatibility with the Amino encoding, due to e.g. hardware
// devices that rely on this encoding.
//
// See CanonicalizeProposal
func ProposalSignBytes(pb *pb.Proposal) []byte {
	bz, err := protoio.MarshalDelimited(pb)
	if err != nil {
		panic(err)
	}

	return bz
}

// ToProto converts Proposal to protobuf
func (p *Proposal) ToProto() *pb.Proposal {
	if p == nil {
		return &pb.Proposal{}
	}
	pb := new(pb.Proposal)

	pb.Type = p.Type
	pb.Round = p.Round
	pb.PolRound = p.POLRound
	pb.Data = p.Data
	pb.Timestamp = p.Timestamp
	pb.Signature = p.Signature
	pb.ChainID = p.ChainID

	return pb
}

// FromProto sets a protobuf Proposal to the given pointer.
// It returns an error if the proposal is invalid.
func ProposalFromProto(pp *pb.Proposal) (*Proposal, error) {
	if pp == nil {
		return nil, errors.New("nil proposal")
	}

	p := new(Proposal)

	p.Type = pp.Type
	p.Round = pp.Round
	p.POLRound = pp.PolRound
	p.Data = pp.Data
	p.Timestamp = pp.Timestamp
	p.Signature = pp.Signature
	p.ChainID = pp.ChainID

	return p, p.ValidateBasic()
}
