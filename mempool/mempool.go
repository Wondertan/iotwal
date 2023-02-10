package mempool

import (
	"context"

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
	pubsub *pubsub.PubSub
	topic  *pubsub.Topic

	txQueue *TxPriorityQueue

	validator ValidateFn

	ctx    context.Context
	cancel context.CancelFunc
}

func NewMempool(h host.Host, validator ValidateFn, c Comparator) (*Mempool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	pubsub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}
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
