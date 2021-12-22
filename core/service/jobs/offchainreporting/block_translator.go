package offchainreporting

import (
	"context"
	"math/big"

	"PhoenixOracle/core/chain"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/db/models"
)

type BlockTranslator interface {
	NumberToQueryRange(ctx context.Context, changedInL1Block uint64) (fromBlock *big.Int, toBlock *big.Int)
}

func NewBlockTranslator(chain *chain.Chain, client ethereum.Client) BlockTranslator {
	if chain == nil {
		return &l1BlockTranslator{}
	} else if chain.IsArbitrum() {
		return NewArbitrumBlockTranslator(client)
	} else if chain.IsOptimism() {
		return newOptimismBlockTranslator()
	}
	return &l1BlockTranslator{}
}

type l1BlockTranslator struct{}

func (*l1BlockTranslator) NumberToQueryRange(_ context.Context, changedInL1Block uint64) (fromBlock *big.Int, toBlock *big.Int) {
	return big.NewInt(int64(changedInL1Block)), big.NewInt(int64(changedInL1Block))
}

func (*l1BlockTranslator) OnNewLongestChain(context.Context, models.Head) {}

type optimismBlockTranslator struct{}

func newOptimismBlockTranslator() *optimismBlockTranslator {
	return &optimismBlockTranslator{}
}

func (*optimismBlockTranslator) NumberToQueryRange(_ context.Context, changedInL1Block uint64) (fromBlock *big.Int, toBlock *big.Int) {
	return big.NewInt(0), nil
}
