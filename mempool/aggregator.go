package mempool

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
)

// TODO: add String()
type namespace int

const (
	Tendermint namespace = 1
)

type aggregator struct {
	mempools     map[namespace]*Mempool
	aggregatorLk sync.RWMutex
}

func NewAggregator() *aggregator {
	return &aggregator{
		mempools: make(map[namespace]*Mempool),
	}
}

func (a *aggregator) Join(ctx context.Context, nId namespace, h host.Host, v ValidateFn, c Comparator) error {
	a.aggregatorLk.Lock()
	defer a.aggregatorLk.Unlock()
	if _, ok := a.mempools[nId]; ok {
		return errors.New("already exist")
	}

	m, err := NewMempool(ctx, h, v, c)
	if err != nil {
		return err
	}
	a.mempools[nId] = m
	return m.Start(context.TODO())
}

func (a *aggregator) Stop(nId namespace) error {
	a.aggregatorLk.Lock()
	a.aggregatorLk.Unlock()
	m, ok := a.mempools[nId]
	if !ok {
		return errors.New("not exist")
	}
	delete(a.mempools, nId)
	return m.Stop(context.TODO())
}

func (a *aggregator) TxByGasLimit(nId namespace, maxGas, maxSize uint64) ([]Tx, error) {
	a.aggregatorLk.Lock()
	a.aggregatorLk.Unlock()
	m, ok := a.mempools[nId]
	if !ok {
		return nil, errors.New("not exist")
	}
	txs := m.ReapMaxBytesMaxGas(maxGas, maxSize)
	if len(txs) == 0 {
		return nil, fmt.Errorf("empty mempool:%d", nId)
	}
	return txs, nil
}

func (a *aggregator) Remove(nId namespace, hashes []string) error {
	a.aggregatorLk.Lock()
	a.aggregatorLk.Unlock()
	m, ok := a.mempools[nId]
	if !ok {
		return errors.New("not exist")
	}
	return m.RemoveTxsByKey(hashes...)
}
