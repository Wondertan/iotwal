package concord

import (
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("conciliator")

type Consensus interface {
	AgreeOn([]byte) ([]byte, *Commit, error)
}
