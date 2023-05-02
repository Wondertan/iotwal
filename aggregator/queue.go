package mempool

import (
	"container/heap"
	"sync"

	"github.com/rs/zerolog/log"

	tx "github.com/Wondertan/iotwal/transaction"
)

var _ heap.Interface = (*TxPriorityQueue)(nil)

// TxPriorityQueue defines a thread-safe priority queue for valid transactions.
type TxPriorityQueue struct {
	sync.RWMutex

	txs     []tx.Tx
	seenTxs map[string]int64 // represents available txs in mempool with their indexes
}

func newTxPriorityQueue() *TxPriorityQueue {
	pq := &TxPriorityQueue{
		txs: make([]tx.Tx, 0),
	}

	heap.Init(pq)
	return pq
}

// numTxs returns the number of transactions in the priority queue. It is
// thread safe.
func (pq *TxPriorityQueue) numTxs() int {
	pq.RLock()
	defer pq.RUnlock()
	return len(pq.txs)
}

// PushTx adds a valid transaction to the priority queue. It is thread safe.
func (pq *TxPriorityQueue) insertTx(tx tx.Tx) bool {
	pq.Lock()
	defer pq.Unlock()

	if i, ok := pq.seenTxs[tx.Hash().String()]; ok && i != -1 { // tx already stored in queue
		return false
	}

	heap.Push(pq, tx)
	return true
}

func (pq *TxPriorityQueue) removeTx(hashes ...string) {
	pq.Lock()
	defer pq.Lock()

	for _, hash := range hashes {
		index, ok := pq.seenTxs[hash]
		if !ok || index == -1 {
			log.Warn()
			continue
		}

		heap.Remove(pq, int(index))
		pq.seenTxs[hash] = -1
	}
}

// popTx removes the top priority transaction from the queue. It is thread safe.
func (pq *TxPriorityQueue) popTx() tx.Tx {
	pq.Lock()
	defer pq.Unlock()

	x := heap.Pop(pq)
	if x != nil {
		return x.(tx.Tx)
	}
	return nil
}

// Push implements the Heap interface.
//
// NOTE: A caller should never call Push. Use PushTx instead.
func (pq *TxPriorityQueue) Push(x any) {
	tx := x.(tx.Tx)
	pq.txs = append(pq.txs, tx)
	pq.seenTxs[tx.Hash().String()] = int64(len(pq.txs) - 1)
}

// Pop implements the Heap interface.
//
// NOTE: A caller should never call Pop. Use PopTx instead.
func (pq *TxPriorityQueue) Pop() any {
	old := pq.txs
	n := len(old)
	tx := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.txs = old[:n-1]
	pq.seenTxs[tx.Hash().String()] = -1
	return tx
}

// Len implements the Heap interface.
//
// NOTE: A caller should never call Len. Use numTxs instead.
func (pq *TxPriorityQueue) Len() int {
	return len(pq.txs)
}

// Less implements the Heap interface. It returns true if the transaction at
// position i in the queue is of less priority than the transaction at position j.
func (pq *TxPriorityQueue) Less(i, j int) bool {
	return pq.txs[i].Fee() < pq.txs[j].Fee()
}

// Swap implements the Heap interface. It swaps two transactions in the queue.
func (pq *TxPriorityQueue) Swap(i, j int) {
	pq.txs[i], pq.txs[j] = pq.txs[j], pq.txs[i]

	pq.seenTxs[pq.txs[i].Hash().String()] = int64(i)
	pq.seenTxs[pq.txs[j].Hash().String()] = int64(j)
}
