package concord

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/Wondertan/iotwal/concord/pb"
)

type RoundVoteSet struct {
	Prevotes   *VoteSet
	Precommits *VoteSet
}

var (
	ErrGotVoteFromUnwantedRound = errors.New(
		"peer has sent a vote that does not match our round for more than one round",
	)
)

/*
HeightVoteSet keeps track of all VoteSets.

A commit is +2/3 precommits for a block at a round,
but which round is not known in advance, so when a peer
provides a precommit for a round greater than mtx.round,
we create a new entry in roundVoteSets but also remember the
peer to prevent abuse.
*/
type HeightVoteSet struct {
	chainID string
	valSet  *ProposerSet

	mtx           sync.Mutex
	roundVoteSets *RoundVoteSet
	// TODO: remove
	peerCatchupRounds map[peer.ID][]int32 // keys: peer.ID; values: at most 2 rounds
}

func NewHeightVoteSet(chainID string, valSet *ProposerSet) *HeightVoteSet {
	return &HeightVoteSet{
		chainID: chainID,
		roundVoteSets: &RoundVoteSet{
			Prevotes:   NewVoteSet(chainID, pb.PrevoteType, valSet),
			Precommits: NewVoteSet(chainID, pb.PrecommitType, valSet),
		},
	}
}

// AddVote duplicate votes return added=false, err=nil.
// By convention, peerID is "" if origin is self.
func (hvs *HeightVoteSet) AddVote(vote *Vote) (added bool, err error) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	if !IsVoteTypeValid(vote.Type) {
		return
	}
	voteSet := hvs.VoteSet(vote.Type)
	added, err = voteSet.AddVote(vote)
	return
}

// POLInfo last round and blockID that has +2/3 prevotes for a particular block or nil.
// Returns -1 if no such round exists.
func (hvs *HeightVoteSet) POLInfo() (polBlockID BlockID) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	rvs := hvs.VoteSet(pb.PrevoteType)
	polBlockID, ok := rvs.TwoThirdsMajority()
	if ok {
		return polBlockID
	}

	return BlockID{}
}

func (hvs *HeightVoteSet) VoteSet(voteType pb.SignedMsgType) *VoteSet {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	switch voteType {
	case pb.PrevoteType:
		return hvs.roundVoteSets.Prevotes
	case pb.PrecommitType:
		return hvs.roundVoteSets.Precommits
	default:
		panic(fmt.Sprintf("Unexpected vote type %X", voteType))
	}
}

// SetPeerMaj23 if a peer claims that it has 2/3 majority for given blockKey, call this.
// NOTE: if there are too many peers, or too much peer churn,
// this can cause memory issues.
// TODO: implement ability to remove peers too
func (hvs *HeightVoteSet) SetPeerMaj23(
	voteType pb.SignedMsgType,
	peerID peer.ID,
	blockID BlockID) error {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	if !IsVoteTypeValid(voteType) {
		return fmt.Errorf("setPeerMaj23: Invalid vote type %X", voteType)
	}
	voteSet := hvs.VoteSet(voteType)
	if voteSet == nil {
		return nil // something we don't know about yet
	}
	return voteSet.SetPeerMaj23(peerID, blockID)
}

//---------------------------------------------------------
// string and json

func (hvs *HeightVoteSet) String() string {
	return hvs.StringIndented("")
}

func (hvs *HeightVoteSet) StringIndented(indent string) string {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	vsStrings := make([]string, 0, 2)
	vsStrings = append(vsStrings, hvs.roundVoteSets.Prevotes.StringShort())
	vsStrings = append(vsStrings, hvs.roundVoteSets.Precommits.StringShort())

	return fmt.Sprintf(`HeightVoteSet{
%s  %v
%s}`,
		indent, strings.Join(vsStrings, "\n"+indent+"  "),
		indent)
}

func (hvs *HeightVoteSet) MarshalJSON() ([]byte, error) {
	hvs.mtx.Lock()
	defer hvs.mtx.Unlock()
	return tmjson.Marshal(roundVotes{
		Prevotes:           hvs.roundVoteSets.Prevotes.VoteStrings(),
		PrevotesBitArray:   hvs.roundVoteSets.Prevotes.BitArrayString(),
		Precommits:         hvs.roundVoteSets.Precommits.VoteStrings(),
		PrecommitsBitArray: hvs.roundVoteSets.Precommits.BitArrayString(),
	})
}

type roundVotes struct {
	Round              int32    `json:"round"`
	Prevotes           []string `json:"prevotes"`
	PrevotesBitArray   string   `json:"prevotes_bit_array"`
	Precommits         []string `json:"precommits"`
	PrecommitsBitArray string   `json:"precommits_bit_array"`
}
