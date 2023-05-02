package mempool

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type mempool struct {
	topic   *pubsub.Topic
	txQueue *TxPriorityQueue

	ctx context.Context
}

func newMempool(ctx context.Context, t *pubsub.Topic) *mempool {
	return &mempool{
		topic:   t,
		txQueue: newTxPriorityQueue(),
		ctx:     ctx,
	}
}

func (p *mempool) subscribe() {
	subscription, err := p.topic.Subscribe()
	if err != nil {
		return
	}
	defer subscription.Cancel()

	for {
		data, err := subscription.Next(p.ctx)
		if err != nil {
			return
		}
		tx, ok := data.ValidatorData.(Tx)
		if !ok {
			panic("invalid data received")
		}
		if ok := p.txQueue.insertTx(tx); !ok {
			// add log here
		}
	}
}

func (p *mempool) remove(txs []Tx) {
	p.txQueue.removeTx(hashes(txs)...)
}

func hashes(txs []Tx) []string {
	h := make([]string, len(txs))
	for i, tx := range txs {
		h[i] = tx.Hash().String()
	}
	return h
}

func (p *mempool) reapMaxBytesMaxGas(ctx context.Context, maxBytes, maxGas uint64) ([]Tx, error) {
	var (
		totalGas  uint64
		totalSize uint64
	)

	// wTxs contains a list of txs retrieved from the priority queue that
	// need to be re-enqueued prior to returning.
	wTxs := make([]Tx, 0, p.txQueue.numTxs())
	defer func() {
		for _, wtx := range wTxs {
			p.txQueue.insertTx(wtx)
		}
	}()

	txs := make([]Tx, 0, p.txQueue.numTxs())
	for p.txQueue.numTxs() > 0 {
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		case <-ctx.Done():
			return nil, p.ctx.Err()
		default:
		}

		tx := p.txQueue.popTx()
		txs = append(txs, tx)
		wTxs = append(wTxs, tx)
		size := tx.Size()

		// Ensure we have capacity for the transaction with respect to the
		// transaction size.
		if totalSize+size > maxBytes {
			return txs[:len(txs)-1], nil
		}

		totalSize += size

		// ensure we have capacity for the transaction with respect to total gas
		gas := totalGas + tx.Gas()
		if gas > maxGas {
			return txs[:len(txs)-1], nil
		}

		totalGas = gas
	}

	return txs, nil
}
