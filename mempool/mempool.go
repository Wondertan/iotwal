package mempool

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
	topicID = "/iotwal/mempool/v0.0.1"
)

// ValidateFn is a client validate func that will request transaction from the network,
// verify its correctness and return a Tx.
type ValidateFn func(context.Context, []byte) (Tx, error)

type Mempool struct {
	mtx sync.RWMutex

	pubsub *pubsub.PubSub
	topic  *pubsub.Topic

	txQueue *TxPriorityQueue

	validator ValidateFn

	ctx    context.Context
	cancel context.CancelFunc
}

func NewMempool(ctx context.Context, h host.Host, validator ValidateFn, c Comparator) (*Mempool, error) {
	pubsub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Mempool{
		pubsub:    pubsub,
		validator: validator,
		txQueue:   NewTxPriorityQueue(c),
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

func (m *Mempool) Start(context.Context) error {
	topic, err := m.pubsub.Join(topicID)
	if err != nil {
		return err
	}
	m.topic = topic
	err = m.pubsub.RegisterTopicValidator(
		topicID,
		m.processIncoming,
	)
	if err != nil {
		return err
	}
	go m.subscribe()
	return nil
}

func (m *Mempool) Stop(context.Context) error {
	err := m.pubsub.UnregisterTopicValidator(topicID)
	if err != nil {
		return err
	}
	err = m.topic.Close()
	if err != nil {
		return err
	}
	m.cancel()
	m.ctx = nil
	return nil
}

func (m *Mempool) subscribe() {
	subsciption, err := m.topic.Subscribe()
	if err != nil {
		return
	}
	defer subsciption.Cancel()
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}
		data, err := subsciption.Next(m.ctx)
		if err != nil {
			return
		}
		tx, ok := data.ValidatorData.(Tx) // TODO: add valid tx to the pool
		if !ok {
			panic("invalid data received")
		}
		if ok := m.txQueue.insertTx(tx); !ok {
			// add log here
		}
	}
}

func (m *Mempool) processIncoming(
	ctx context.Context,
	from peer.ID,
	msg *pubsub.Message,
) pubsub.ValidationResult {
	tx, err := m.validator(ctx, msg.Data)
	if err != nil {
		m.pubsub.BlacklistPeer(from)
		return pubsub.ValidationReject
	}
	if !tx.ValidateBasic() {
		m.pubsub.BlacklistPeer(from)
		return pubsub.ValidationReject
	}

	msg.ValidatorData = tx
	return pubsub.ValidationAccept
}

func (m *Mempool) RemoveTxsByKey(hashes ...string) error {
	for _, hash := range hashes {
		if !m.txQueue.removeTx(hash) {
			// TODO: append errors after updating to golang 1.20
			// and notify what tx we were not able to delete
		}
	}
	return nil
}

// ReapMaxBytesMaxGas returns a list of transactions within the provided size
// and gas constraints. Transaction are retrieved in priority order.
//
// NOTE:
// - Transactions returned are not removed from the mempool transaction store or indexes.
func (m *Mempool) ReapMaxBytesMaxGas(maxBytes, maxGas uint64) []Tx {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	var (
		totalGas  uint64
		totalSize uint64
	)

	// wTxs contains a list of *WrappedTx retrieved from the priority queue that
	// need to be re-enqueued prior to returning.
	wTxs := make([]Tx, 0, m.txQueue.numTxs())
	defer func() {
		for _, wtx := range wTxs {
			m.txQueue.insertTx(wtx)
		}
	}()

	txs := make([]Tx, 0, m.txQueue.numTxs())
	for m.txQueue.numTxs() > 0 {
		tx := m.txQueue.popTx()
		txs = append(txs, tx)
		wTxs = append(wTxs, tx)
		size := tx.Size()

		// Ensure we have capacity for the transaction with respect to the
		// transaction size.
		if maxBytes > -1 && totalSize+size > maxBytes {
			return txs[:len(txs)-1]
		}

		totalSize += size

		// ensure we have capacity for the transaction with respect to total gas
		gas := totalGas + tx.Gas()
		if maxGas > -1 && gas > maxGas {
			return txs[:len(txs)-1]
		}

		totalGas = gas
	}

	return txs
}
