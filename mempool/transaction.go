package mempool

// Tx represents a read-only interface for the transactions.
type Tx interface {
	Hash() []byte
	Sender() []byte
	Priority() uint64
	Size() uint64
	Gas() uint64
}
