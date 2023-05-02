package mempool

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	tx "github.com/Wondertan/iotwal/transaction"
)

const (
	topicID = "/aggregator/v0.0.1"
)

// ValidateFn is a client validate func that will request transaction from the network,
// verify its correctness and return a Tx.
type ValidateFn func(context.Context, []byte) (tx.Tx, error)

type Aggregator struct {
	mtx sync.RWMutex

	pubsub *pubsub.PubSub

	mempools map[string]*mempool

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAggregator(ctx context.Context, h host.Host) (*Aggregator, error) {
	pubsub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Aggregator{
		pubsub:   pubsub,
		mempools: make(map[string]*mempool),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (a *Aggregator) Join(networkID string, validator ValidateFn) error {
	topicID := pubsubTopicID(networkID)

	topic, err := a.pubsub.Join(topicID)
	if err != nil {
		return err
	}

	if err = a.pubsub.RegisterTopicValidator(
		topicID,
		func(
			ctx context.Context,
			from peer.ID,
			msg *pubsub.Message,
		) pubsub.ValidationResult {
			tx, err := validator(ctx, msg.Data)
			if err != nil {
				a.pubsub.BlacklistPeer(from)
				return pubsub.ValidationReject
			}
			if !tx.ValidateBasic() {
				a.pubsub.BlacklistPeer(from)
				return pubsub.ValidationReject
			}

			msg.ValidatorData = tx
			return pubsub.ValidationAccept
		},
	); err != nil {
		return err
	}

	pool := newMempool(a.ctx, topic)

	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.mempools[networkID] = pool

	go pool.subscribe()
	return nil
}

func (a *Aggregator) Leave(networkID string) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.leave(networkID)
}

func (a *Aggregator) leave(networkID string) error {
	pool, ok := a.mempools[networkID]
	if !ok {
		return errors.New("pool was not registered")
	}

	err := a.pubsub.UnregisterTopicValidator(pool.topic.String())
	if err != nil {
		return err
	}

	err = pool.topic.Close()
	if err != nil {
		return err
	}

	delete(a.mempools, networkID)
	return nil
}

func (a *Aggregator) Stop() error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	var multiErr error
	for networkID := range a.mempools {
		err := a.leave(networkID)
		if err != nil {
			multiErr = multierror.Append(err, fmt.Errorf("could not delete pool:%s, %v", networkID, err))
		}
	}
	return multiErr
}

func (a *Aggregator) RemoveTxs(networkID string, txs []tx.Tx) error {
	a.mtx.RLock()
	defer a.mtx.RUnlock()

	pool, ok := a.mempools[networkID]
	if !ok {

		return errors.New("pool not registered")
	}

	go pool.remove(txs)
	return nil
}

// ReapMaxBytesMaxGas returns a list of transactions within the provided size
// and gas constraints. Transaction are retrieved in priority order.
//
// NOTE:
// - Transactions returned are not removed from the Aggregator transaction store or indexes.
func (a *Aggregator) ReapMaxBytesMaxGas(ctx context.Context, networkID string, maxbytes, maxGas uint64) (<-chan []tx.Tx, error) {
	a.mtx.RLock()
	defer a.mtx.RUnlock()

	pool, ok := a.mempools[networkID]
	if !ok {
		return nil, errors.New("pool not registered")
	}

	txsCh := make(chan []tx.Tx)

	go func() {
		txs, err := pool.reapMaxBytesMaxGas(ctx, maxbytes, maxGas)
		if err != nil {
			// TODO: add logs
		}
		txsCh <- txs
		close(txsCh)
	}()
	return txsCh, nil
}

func pubsubTopicID(networkID string) string {
	return fmt.Sprintf("/%s%s", networkID, topicID)
}
