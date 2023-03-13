package mempool

import (
	"container/heap"
	"sync"
)

var _ heap.Interface = (*TxPriorityQueue)(nil)

type Comparator func(Tx, Tx) bool

// TxPriorityQueue defines a thread-safe priority queue for valid transactions.
type TxPriorityQueue struct {
	sync.RWMutex

	txs     []Tx
	seenTxs map[string]int64 // represents available txs in mempool with their indexes
}

func newTxPriorityQueue() *TxPriorityQueue {
	pq := &TxPriorityQueue{
		txs: make([]Tx, 0),
	}

	heap.Init(pq)

	return pq
}

// NumTxs returns the number of transactions in the priority queue. It is
// thread safe.
func (pq *TxPriorityQueue) numTxs() int {
	pq.RLock()
	defer pq.RUnlock()

	return len(pq.txs)
}

// PushTx adds a valid transaction to the priority queue. It is thread safe.
func (pq *TxPriorityQueue) insertTx(tx Tx) bool {
	pq.Lock()
	defer pq.Unlock()

	if i, ok := pq.seenTxs[string(tx.Hash())]; ok && i != -1 { // tx already stored in queue
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
			// TODO: add log that tx not exist
			continue
		}

		heap.Remove(pq, int(index))
		pq.seenTxs[hash] = -1
	}
}

// popTx removes the top priority transaction from the queue. It is thread safe.
func (pq *TxPriorityQueue) popTx() Tx {
	pq.Lock()
	defer pq.Unlock()

	x := heap.Pop(pq)
	if x != nil {
		return x.(Tx)
	}
	return nil
}

// Push implements the Heap interface.
//
// NOTE: A caller should never call Push. Use PushTx instead.
func (pq *TxPriorityQueue) Push(x any) {
	tx := x.(Tx)
	pq.txs = append(pq.txs, tx)
	pq.seenTxs[string(tx.Hash())] = int64(len(pq.txs) - 1)
}

// Pop implements the Heap interface.
//
// NOTE: A caller should never call Pop. Use PopTx instead.
func (pq *TxPriorityQueue) Pop() any {
	old := pq.txs
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.txs = old[:n-1]
	pq.seenTxs[string(item.Hash())] = -1
	return item
}

// Len implements the Heap interface.
//
// NOTE: A caller should never call Len. Use NumTxs instead.
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

	pq.seenTxs[string(pq.txs[i].Hash())] = int64(i)
	pq.seenTxs[string(pq.txs[j].Hash())] = int64(j)
}
