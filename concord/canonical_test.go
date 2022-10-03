package concord

import (
	"reflect"
	"testing"

	"github.com/Wondertan/iotwal/concord/pb"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

func TestCanonicalizeBlockID(t *testing.T) {
	randhash := tmrand.Bytes(tmhash.Size)
	block1 := pb.BlockID{Hash: randhash}
	block2 := pb.BlockID{Hash: randhash}
	cblock1 := pb.CanonicalBlockID{Hash: randhash}
	cblock2 := pb.CanonicalBlockID{Hash: randhash}

	tests := []struct {
		name string
		args pb.BlockID
		want *pb.CanonicalBlockID
	}{
		{"first", block1, &cblock1},
		{"second", block2, &cblock2},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := CanonicalizeBlockID(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CanonicalizeBlockID() = %v, want %v", got, tt.want)
			}
		})
	}
}
