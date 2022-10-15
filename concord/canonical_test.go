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
	block1 := pb.DataHash{Hash: randhash}
	block2 := pb.DataHash{Hash: randhash}
	cblock1 := pb.CanonicalDataHash{Hash: randhash}
	cblock2 := pb.CanonicalDataHash{Hash: randhash}

	tests := []struct {
		name string
		args pb.DataHash
		want *pb.CanonicalDataHash
	}{
		{"first", block1, &cblock1},
		{"second", block2, &cblock2},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := CanonicalizeDataHash(&tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CanonicalizeBlockID() = %v, want %v", got, tt.want)
			}
		})
	}
}
