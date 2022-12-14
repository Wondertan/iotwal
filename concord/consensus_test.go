package concord

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"golang.org/x/sync/errgroup"
)

func createP2PNetwork(t *testing.T, amount int) []*pubsub.PubSub {
	mNet := createMocknet(t, amount)
	return createPubSub(t, mNet.Hosts())
}

func createMocknet(t *testing.T, amount int) mocknet.Mocknet {
	require.True(t, amount > 0)
	net, err := mocknet.FullMeshConnected(amount)
	require.NoError(t, err)
	return net
}

func createPubSub(t *testing.T, hosts []host.Host) []*pubsub.PubSub {
	psubs := make([]*pubsub.PubSub, 0, len(hosts))

	for _, host := range hosts {
		ps, err := pubsub.NewFloodSub(context.Background(), host, pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign))
		require.NoError(t, err)
		psubs = append(psubs, ps)
	}

	return psubs
}

func Test_AgreeOn(t *testing.T) {
	pSubs := createP2PNetwork(t, 5)
	valSet, privValidators := RandValidatorSet(5, 10)
	concords := make([]Consensus, 0, len(pSubs))
	for i, pSub := range pSubs {
		conciliator := NewConciliator(pSub, privValidators[i])
		concord, err := conciliator.NewConcord("iotwal", func(ctx context.Context, data []byte) (tmbytes.HexBytes, error) {
			return data, nil
		})
		require.NoError(t, err)
		concords = append(concords, concord)
	}

	errgrp, ctx := errgroup.WithContext(context.Background())
	results := make([][]byte, len(pSubs))
	commits := make([]*Commit, len(pSubs))
	for i, c := range concords {
		c := c
		i := i
		pubKey, err := privValidators[i].GetPubKey()
		require.NoError(t, err)
		errgrp.Go(func() error {
			data, commit, err := c.AgreeOn(ctx, pubKey.Bytes(), valSet)
			results[i] = data

			commits[i] = commit
			return err
		})
	}

	err := errgrp.Wait()
	require.NoError(t, err)
	for index := 1; index < len(results); index++ {
		require.Equal(t, results[0], results[index])
	}

	for _, commit := range commits {
		err := valSet.VerifyCommit("iotwal", results[0], commit)
		require.NoError(t, err)
	}
}
