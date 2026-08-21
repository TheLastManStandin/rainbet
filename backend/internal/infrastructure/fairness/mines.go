package fairness

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"

	"rainbet/internal/domain/mines"
)

const uint32Range = uint64(1) << 32

type Generator struct{}

func (Generator) Indexes(game *mines.Game) ([]int, error) {
	return DetermineMineIndexes(Options{
		Tiles:             game.GridSize,
		Mines:             game.Mines,
		ClientSeed:        game.ClientSeed,
		ServerSeed:        game.ServerSeed,
		TransactionNumber: game.Nonce,
	})
}

type Options struct {
	Tiles             int
	Mines             int
	ClientSeed        string
	ServerSeed        string
	TransactionNumber int64
}

type uint32Stream struct {
	key     []byte
	message string
	step    uint64
	values  [sha512.Size / 4]uint32
	index   int
}

func GenerateServerSeed() (string, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", fmt.Errorf("generate server seed: %w", err)
	}
	return hex.EncodeToString(seed), nil
}

func DetermineMineIndexes(options Options) ([]int, error) {
	if options.Tiles <= 0 || uint64(options.Tiles) > uint32Range {
		return nil, fmt.Errorf("invalid tile count")
	}
	if options.Mines < 0 || options.Mines > options.Tiles {
		return nil, fmt.Errorf("invalid mine count")
	}
	if options.ClientSeed == "" {
		return nil, fmt.Errorf("client seed is required")
	}
	if options.ServerSeed == "" {
		return nil, fmt.Errorf("server seed is required")
	}
	if options.TransactionNumber < 0 {
		return nil, fmt.Errorf("invalid transaction number")
	}

	indexes := make([]int, options.Tiles)
	for i := range indexes {
		indexes[i] = i
	}
	message := options.ClientSeed + ":" + strconv.FormatInt(options.TransactionNumber, 10) + ":mines"
	return drawWithoutReplacement(newUint32Stream(options.ServerSeed, message), indexes, options.Mines)
}

func newUint32Stream(key, message string) *uint32Stream {
	return &uint32Stream{key: []byte(key), message: message, index: sha512.Size / 4}
}

func (stream *uint32Stream) next() uint32 {
	if stream.index == len(stream.values) {
		mac := hmac.New(sha512.New, stream.key)
		_, _ = mac.Write([]byte(stream.message + ":" + strconv.FormatUint(stream.step, 10)))
		hash := mac.Sum(nil)
		for i := range stream.values {
			stream.values[i] = binary.BigEndian.Uint32(hash[i*4 : i*4+4])
		}
		stream.step++
		stream.index = 0
	}
	value := stream.values[stream.index]
	stream.index++
	return value
}

func drawWithoutReplacement(stream *uint32Stream, values []int, count int) ([]int, error) {
	result := append([]int(nil), values...)
	for i := len(result) - 1; i >= len(result)-count; i-- {
		index, err := unbiasedInt(stream, i+1)
		if err != nil {
			return nil, err
		}
		result[i], result[index] = result[index], result[i]
	}
	return result[len(result)-count:], nil
}

func unbiasedInt(stream *uint32Stream, upperBound int) (int, error) {
	if upperBound <= 0 || uint64(upperBound) > uint32Range {
		return 0, fmt.Errorf("invalid upper bound")
	}
	maxAcceptable := uint32Range - uint32Range%uint64(upperBound)
	for {
		value := uint64(stream.next())
		if value < maxAcceptable {
			return int(value % uint64(upperBound)), nil
		}
	}
}
