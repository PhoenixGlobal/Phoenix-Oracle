package cryptotest

import (
	"math/rand"
	"testing"
)

type randomStream rand.Rand

func NewStream(t *testing.T, seed int64) *randomStream {
	return (*randomStream)(rand.New(rand.NewSource(seed)))
}

func (s *randomStream) XORKeyStream(dst, src []byte) {
	(*rand.Rand)(s).Read(dst)
}
