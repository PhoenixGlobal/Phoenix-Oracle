package proof

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"PhoenixOracle/util"
)

type Seed [32]byte

func BigToSeed(x *big.Int) (Seed, error) {
	seed, err := utils.Uint256ToBytes(x)
	if err != nil {
		return Seed{}, err
	}
	return Seed(common.BytesToHash(seed)), nil
}

func (s *Seed) Big() *big.Int {
	return common.Hash(*s).Big()
}

func BytesToSeed(b []byte) (*Seed, error) {
	if len(b) > 32 {
		return nil, errors.Errorf("Seed representation can be at most 32 bytes, "+
			"got %d", len(b))
	}
	seed := Seed(common.BytesToHash(b))
	return &seed, nil
}

type PreSeedData struct {
	PreSeed   Seed
	BlockHash common.Hash
	BlockNum  uint64
}

type PreSeedDataV2 struct {
	PreSeed          Seed
	BlockHash        common.Hash
	BlockNum         uint64
	SubId            uint64
	CallbackGasLimit uint32
	NumWords         uint32
	Sender           common.Address
}

func FinalSeed(s PreSeedData) (finalSeed *big.Int) {
	seedHashMsg := append(s.PreSeed[:], s.BlockHash.Bytes()...)
	return utils.MustHash(string(seedHashMsg)).Big()
}

func FinalSeedV2(s PreSeedDataV2) (finalSeed *big.Int) {
	seedHashMsg := append(s.PreSeed[:], s.BlockHash.Bytes()...)
	return utils.MustHash(string(seedHashMsg)).Big()
}

func TestXXXSeedData(t *testing.T, preSeed *big.Int, blockHash common.Hash,
	blockNum int) PreSeedData {
	seedAsSeed, err := BigToSeed(big.NewInt(0x10))
	require.NoError(t, err, "seed %x out of range", 0x10)
	return PreSeedData{
		PreSeed:   seedAsSeed,
		BlockNum:  uint64(blockNum),
		BlockHash: blockHash,
	}
}
