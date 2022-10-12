package concord

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/Wondertan/iotwal/concord/pb"
)

type DataHash struct {
	Hash tmbytes.HexBytes `json:"hash"`
}

// Equals returns true if the BlockID matches the given BlockID
func (dataHash DataHash) Equals(other DataHash) bool {
	return bytes.Equal(dataHash.Hash, other.Hash)
}

// Key returns a machine-readable string representation of the BlockID
func (dataHash DataHash) Key() string {
	return string(dataHash.Hash)
}

// ValidateBasic performs basic validation.
func (dataHash DataHash) ValidateBasic() error {
	// Hash can be empty in case of POLBlockID in Proposal.
	if err := ValidateHash(dataHash.Hash); err != nil {
		return fmt.Errorf("wrong Hash")
	}
	return nil
}

// IsZero returns true if this is the BlockID of a nil block.
func (dataHash DataHash) IsZero() bool {
	return len(dataHash.Hash) == 0
}

// IsComplete returns true if this is a valid BlockID of a non-nil block.
func (dataHash DataHash) IsComplete() bool {
	return len(dataHash.Hash) == tmhash.Size
}

func DataHashFromProto(bID *pb.DataHash) (*DataHash, error) {
	if bID == nil {
		return nil, errors.New("nil BlockID")
	}

	blockID := new(DataHash)
	blockID.Hash = bID.Hash

	return blockID, blockID.ValidateBasic()
}

// ToProto converts BlockID to protobuf
func (dataHash *DataHash) ToProto() *pb.DataHash {
	if dataHash == nil {
		return &pb.DataHash{}
	}

	return &pb.DataHash{
		Hash: dataHash.Hash,
	}
}
