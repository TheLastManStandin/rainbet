package fairness

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestDetermineMineIndexesMatchesReferenceImplementation(t *testing.T) {
	indexes, err := DetermineMineIndexes(Options{
		Tiles: 25, Mines: 10, ClientSeed: "mine-test", ServerSeed: "mine-test", TransactionNumber: 1,
	})
	if err != nil {
		t.Fatalf("determine mine indexes: %v", err)
	}
	want := []int{15, 6, 5, 12, 9, 23, 14, 2, 8, 19}
	if !reflect.DeepEqual(indexes, want) {
		t.Fatalf("indexes = %v, want %v", indexes, want)
	}
}

func TestGenerateServerSeed(t *testing.T) {
	seed, err := GenerateServerSeed()
	if err != nil {
		t.Fatalf("generate server seed: %v", err)
	}
	decoded, err := hex.DecodeString(seed)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("seed = %q, decoded bytes = %d, error = %v", seed, len(decoded), err)
	}
}

func TestDetermineMineIndexesRejectsInvalidOptions(t *testing.T) {
	_, err := DetermineMineIndexes(Options{Tiles: 25, Mines: 26, ClientSeed: "client", ServerSeed: "server"})
	if err == nil {
		t.Fatal("invalid mine count was accepted")
	}
}
