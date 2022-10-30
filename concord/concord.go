package concord

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("conciliator")

type Consensus interface {
	AgreeOn(context.Context, []byte, *ProposerSet) ([]byte, *Commit, error)
}
