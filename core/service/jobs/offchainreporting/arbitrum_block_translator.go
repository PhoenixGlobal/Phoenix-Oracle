package offchainreporting

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
	"github.com/pkg/errors"
)

type ArbitrumBlockTranslator struct {
	ethClient ethereum.Client

	cache   map[int64]int64
	cacheMu sync.RWMutex
	l2Locks utils.KeyedMutex
}

func NewArbitrumBlockTranslator(ethClient ethereum.Client) *ArbitrumBlockTranslator {
	return &ArbitrumBlockTranslator{
		ethClient,
		make(map[int64]int64),
		sync.RWMutex{},
		utils.KeyedMutex{},
	}
}

func (a *ArbitrumBlockTranslator) NumberToQueryRange(ctx context.Context, changedInL1Block uint64) (fromBlock *big.Int, toBlock *big.Int) {
	var err error
	fromBlock, toBlock, err = a.BinarySearch(ctx, int64(changedInL1Block))
	if err != nil {
		logger.Warnw("ArbitrumBlockTranslator: failed to binary search L2->L1, falling back to slow scan over entire chain", "err", err)
		return big.NewInt(0), nil
	}

	return
}

func (a *ArbitrumBlockTranslator) BinarySearch(ctx context.Context, targetL1 int64) (l2lowerBound *big.Int, l2upperBound *big.Int, err error) {
	mark := time.Now()
	var n int
	defer func() {
		duration := time.Since(mark)
		logger.Debugw(fmt.Sprintf("ArbitrumBlockTranslator#binarySearch completed in %s with %d total lookups", duration, n), "finishedIn", duration, "err", err, "nLookups", n)
	}()
	var h *models.Head

	var l2lower int64
	var l2upper int64

	var skipUpperBound bool

	{
		var maybeL2Upper *int64
		l2lower, maybeL2Upper = a.reverseLookup(targetL1)
		if maybeL2Upper != nil {
			l2upper = *maybeL2Upper
		} else {
			h, err = a.ethClient.HeadByNumber(ctx, nil)
			n++
			if err != nil {
				return nil, nil, err
			}
			if h == nil {
				return nil, nil, errors.New("got nil head")
			}
			if !h.L1BlockNumber.Valid {
				return nil, nil, errors.New("head was missing L1 block number")
			}
			currentL1 := h.L1BlockNumber.Int64
			currentL2 := h.Number

			a.cachePut(currentL2, currentL1)

			if targetL1 > currentL1 {
				logger.Debugf("ArbitrumBlockTranslator#BinarySearch target of %d is above current L1 block number of %d, using nil for upper bound", targetL1, currentL1)
				return big.NewInt(currentL2), nil, nil
			} else if targetL1 == currentL1 {
				skipUpperBound = true
			}
			l2upper = currentL2
		}
	}

	logger.Tracef("ArbitrumBlockTranslator#BinarySearch starting search for L2 range wrapping L1 block number %d between bounds [%d, %d]", targetL1, l2lower, l2upper)

	var exactMatch bool

	{
		l2lower, err = search(l2lower, l2upper+1, func(l2 int64) (bool, error) {
			l1, miss, err2 := a.arbL2ToL1(ctx, l2)
			if miss {
				n++
			}
			if err2 != nil {
				return false, err2
			}
			if targetL1 == l1 {
				exactMatch = true
			}
			return l1 >= targetL1, nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if !skipUpperBound {
		var r int64
		r, err = search(l2lower, l2upper+1, func(l2 int64) (bool, error) {
			l1, miss, err2 := a.arbL2ToL1(ctx, l2)
			if miss {
				n++
			}
			if err2 != nil {
				return false, err2
			}
			if targetL1 == l1 {
				exactMatch = true
			}
			return l1 > targetL1, nil
		})
		if err != nil {
			return nil, nil, err
		}
		l2upper = r - 1
		l2upperBound = big.NewInt(l2upper)
	}

	if !exactMatch {
		return nil, nil, errors.Errorf("target L1 block number %d is not represented by any L2 block", targetL1)
	}
	return big.NewInt(l2lower), l2upperBound, nil
}

func (a *ArbitrumBlockTranslator) reverseLookup(targetL1 int64) (from int64, to *int64) {
	type val struct {
		l1 int64
		l2 int64
	}
	vals := make([]val, 0)

	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	for l2, l1 := range a.cache {
		vals = append(vals, val{l1, l2})
	}

	sort.Slice(vals, func(i, j int) bool { return vals[i].l1 < vals[j].l1 })

	for _, val := range vals {
		if val.l1 < targetL1 {
			from = val.l2
		} else if val.l1 > targetL1 && to == nil {
			l2 := val.l2
			to = &l2
		}
	}
	return
}

func (a *ArbitrumBlockTranslator) arbL2ToL1(ctx context.Context, l2 int64) (l1 int64, cacheMiss bool, err error) {
	unlock := a.l2Locks.LockInt64(l2)
	defer unlock()

	var exists bool
	if l1, exists = a.cacheGet(l2); exists {
		return l1, false, err
	}

	h, err := a.ethClient.HeadByNumber(ctx, big.NewInt(l2))
	if err != nil {
		return 0, true, err
	}
	if h == nil {
		return 0, true, errors.New("got nil head")
	}
	if !h.L1BlockNumber.Valid {
		return 0, true, errors.New("head was missing L1 block number")
	}
	l1 = h.L1BlockNumber.Int64

	a.cachePut(l2, l1)

	return l1, true, nil
}

func (a *ArbitrumBlockTranslator) cacheGet(l2 int64) (l1 int64, exists bool) {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()
	l1, exists = a.cache[l2]
	return
}

func (a *ArbitrumBlockTranslator) cachePut(l2, l1 int64) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	a.cache[l2] = l1
}

func search(i, j int64, f func(int64) (bool, error)) (int64, error) {
	for i < j {
		h := int64(uint64(i+j) >> 1)
		// i ≤ h < j
		is, err := f(h)
		if err != nil {
			return 0, err
		}
		if !is {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	return i, nil
}
