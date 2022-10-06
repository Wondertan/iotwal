package concord

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Wondertan/iotwal/concord/pb"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

type BlockID struct {
	Hash          tmbytes.HexBytes `json:"hash"`
}

// Equals returns true if the BlockID matches the given BlockID
func (blockID BlockID) Equals(other BlockID) bool {
	return bytes.Equal(blockID.Hash, other.Hash)
}

// Key returns a machine-readable string representation of the BlockID
func (blockID BlockID) Key() string {
	return string(blockID.Hash)
}

// ValidateBasic performs basic validation.
func (blockID BlockID) ValidateBasic() error {
	// Hash can be empty in case of POLBlockID in Proposal.
	if err := ValidateHash(blockID.Hash); err != nil {
		return fmt.Errorf("wrong Hash")
	}
	return nil
}

// IsZero returns true if this is the BlockID of a nil block.
func (blockID BlockID) IsZero() bool {
	return len(blockID.Hash) == 0
}

// IsComplete returns true if this is a valid BlockID of a non-nil block.
func (blockID BlockID) IsComplete() bool {
	return len(blockID.Hash) == tmhash.Size
}

// String returns a human readable string representation of the BlockID.
//
// 1. hash
// 2. part set header
//
// See PartSetHeader#String
func (blockID BlockID) String() string {
	return fmt.Sprintf(`%v:`, blockID.Hash)
}

// ToProto converts BlockID to protobuf
func (blockID *BlockID) ToProto() pb.BlockID {
	if blockID == nil {
		return pb.BlockID{}
	}

	return pb.BlockID{
		Hash:          blockID.Hash,
	}
}

// FromProto sets a protobuf BlockID to the given pointer.
// It returns an error if the block id is invalid.
func BlockIDFromProto(bID *pb.BlockID) (*BlockID, error) {
	if bID == nil {
		return nil, errors.New("nil BlockID")
	}

	blockID := new(BlockID)
	blockID.Hash = bID.Hash

	return blockID, blockID.ValidateBasic()
}
