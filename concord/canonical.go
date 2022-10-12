package concord

import (
	"time"

	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/Wondertan/iotwal/concord/pb"
)

// Canonical* wraps the structs in types for amino encoding them for use in SignBytes / the Signable interface.

// TimeFormat is used for generating the sigs
const TimeFormat = time.RFC3339Nano

//-----------------------------------
// Canonicalize the structs

func CanonicalizeDataHash(bid *pb.DataHash) *pb.CanonicalDataHash {
	rbid, err := DataHashFromProto(bid)
	if err != nil {
		panic(err)
	}
	var cbid *pb.CanonicalDataHash
	if rbid == nil || rbid.IsZero() {
		cbid = nil
	} else {
		cbid = &pb.CanonicalDataHash{
			Hash: bid.Hash,
		}
	}

	return cbid
}

// CanonicalizeVote transforms the given Proposal to a CanonicalProposal.
func CanonicalizeProposal(chainID string, proposal *pb.Proposal) pb.CanonicalProposal {
	return pb.CanonicalProposal{
		Type:      pb.ProposalType,
		DataHash:  CanonicalizeDataHash(proposal.DataHash),
		Timestamp: proposal.Timestamp,
		ChainID:   chainID,
	}
}

// CanonicalizeVote transforms the given Vote to a CanonicalVote, which does
// not contain ValidatorIndex and ValidatorAddress fields.
func CanonicalizeVote(chainID string, vote *pb.Vote) pb.CanonicalVote {
	return pb.CanonicalVote{
		Type:      vote.Type,
		Height:    vote.Height,       // encoded as sfixed64
		Round:     int64(vote.Round), // encoded as sfixed64
		DataHash:  CanonicalizeDataHash(&vote.DataHash),
		Timestamp: vote.Timestamp,
		ChainID:   chainID,
	}
}

// CanonicalTime can be used to stringify time in a canonical way.
func CanonicalTime(t time.Time) string {
	// Note that sending time over amino resets it to
	// local time, we need to force UTC here, so the
	// signatures match
	return tmtime.Canonical(t).Format(TimeFormat)
}
