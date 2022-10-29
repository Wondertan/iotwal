package concord

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("conciliator")

type Consensus interface {
	AgreeOn(context.Context, *ProposerSet, []byte) ([]byte, *Commit, error)
}
