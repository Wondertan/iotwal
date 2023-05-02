package tx

import "github.com/ipfs/go-cid"

// Tx represents a read-only interface for the transactions.
type Tx interface {
	ValidateBasic() bool
	Hash() cid.Cid
	Sender() []byte
	Priority() uint64
	Size() uint64
	Gas() uint64
	Fee() uint64
}
