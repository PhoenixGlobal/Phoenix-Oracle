package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"PhoenixOracle/lib/null"
	"PhoenixOracle/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type Head struct {
	ID            uint64
	Hash          common.Hash
	Number        int64
	L1BlockNumber null.Int64
	ParentHash    common.Hash
	Parent        *Head `gorm:"-"`
	Timestamp     time.Time
	CreatedAt     time.Time
}

func NewHead(number *big.Int, blockHash common.Hash, parentHash common.Hash, timestamp uint64) Head {
	return Head{
		Number:     number.Int64(),
		Hash:       blockHash,
		ParentHash: parentHash,
		Timestamp:  time.Unix(int64(timestamp), 0),
	}
}

func (h *Head) EarliestInChain() Head {
	for {
		if h.Parent != nil {
			h = h.Parent
		} else {
			break
		}
	}
	return *h
}

func (h *Head) IsInChain(blockHash common.Hash) bool {
	for {
		if h.Hash == blockHash {
			return true
		}
		if h.Parent != nil {
			h = h.Parent
		} else {
			break
		}
	}
	return false
}

func (h *Head) HashAtHeight(blockNum int64) common.Hash {
	for {
		if h.Number == blockNum {
			return h.Hash
		}
		if h.Parent != nil {
			h = h.Parent
		} else {
			break
		}
	}
	return common.Hash{}
}

func (h *Head) ChainLength() uint32 {
	l := uint32(1)

	for {
		if h.Parent != nil {
			l++
			h = h.Parent
		} else {
			break
		}
	}
	return l
}

func (h *Head) ChainHashes() []common.Hash {
	var hashes []common.Hash

	for {
		hashes = append(hashes, h.Hash)
		if h.Parent != nil {
			h = h.Parent
		} else {
			break
		}
	}
	return hashes
}

func (h *Head) String() string {
	return h.ToInt().String()
}

func (h *Head) ToInt() *big.Int {
	if h == nil {
		return nil
	}
	return big.NewInt(h.Number)
}

func (h *Head) GreaterThan(r *Head) bool {
	if h == nil {
		return false
	}
	if h != nil && r == nil {
		return true
	}
	return h.Number > r.Number
}

func (h *Head) NextInt() *big.Int {
	if h == nil {
		return nil
	}
	return new(big.Int).Add(h.ToInt(), big.NewInt(1))
}

func (h *Head) UnmarshalJSON(bs []byte) error {
	type head struct {
		Hash          common.Hash    `json:"hash"`
		Number        *hexutil.Big   `json:"number"`
		ParentHash    common.Hash    `json:"parentHash"`
		Timestamp     hexutil.Uint64 `json:"timestamp"`
		L1BlockNumber *hexutil.Big   `json:"l1BlockNumber"`
	}

	var jsonHead head
	err := json.Unmarshal(bs, &jsonHead)
	if err != nil {
		return err
	}

	if jsonHead.Number == nil {
		*h = Head{}
		return nil
	}

	h.Hash = jsonHead.Hash
	h.Number = (*big.Int)(jsonHead.Number).Int64()
	h.ParentHash = jsonHead.ParentHash
	h.Timestamp = time.Unix(int64(jsonHead.Timestamp), 0).UTC()
	if jsonHead.L1BlockNumber != nil {
		h.L1BlockNumber = null.Int64From((*big.Int)(jsonHead.L1BlockNumber).Int64())
	}
	return nil
}

func (h *Head) MarshalJSON() ([]byte, error) {
	type head struct {
		Hash       *common.Hash    `json:"hash,omitempty"`
		Number     *hexutil.Big    `json:"number,omitempty"`
		ParentHash *common.Hash    `json:"parentHash,omitempty"`
		Timestamp  *hexutil.Uint64 `json:"timestamp,omitempty"`
	}

	var jsonHead head
	if h.Hash != (common.Hash{}) {
		jsonHead.Hash = &h.Hash
	}
	jsonHead.Number = (*hexutil.Big)(big.NewInt(int64(h.Number)))
	if h.ParentHash != (common.Hash{}) {
		jsonHead.ParentHash = &h.ParentHash
	}
	if h.Timestamp != (time.Time{}) {
		t := hexutil.Uint64(h.Timestamp.UTC().Unix())
		jsonHead.Timestamp = &t
	}
	return json.Marshal(jsonHead)
}

// WeiPerEth is amount of Wei currency units in one Eth.
var WeiPerEth = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

var emptyHash = common.Hash{}

func ReceiptIsUnconfirmed(txr *types.Receipt) bool {
	return txr == nil || txr.TxHash == emptyHash || txr.BlockNumber == nil
}

var PhoenixFulfilledTopic = utils.MustHash("PhoenixFulfilled(bytes32)")

func ReceiptIndicatesRunLogFulfillment(txr types.Receipt) bool {
	for _, log := range txr.Logs {
		if log.Topics[0] == PhoenixFulfilledTopic {
			return true
		}
	}
	return false
}

type FunctionSelector [FunctionSelectorLength]byte

// FunctionSelectorLength should always be a length of 4 as a byte.
const FunctionSelectorLength = 4

// BytesToFunctionSelector converts the given bytes to a FunctionSelector.
func BytesToFunctionSelector(b []byte) FunctionSelector {
	var f FunctionSelector
	f.SetBytes(b)
	return f
}

func HexToFunctionSelector(s string) FunctionSelector {
	return BytesToFunctionSelector(common.FromHex(s))
}

func (f FunctionSelector) String() string { return hexutil.Encode(f[:]) }

func (f FunctionSelector) Bytes() []byte { return f[:] }

func (f *FunctionSelector) SetBytes(b []byte) { copy(f[:], b[:FunctionSelectorLength]) }

var hexRegexp = regexp.MustCompile("^[0-9a-fA-F]*$")

func unmarshalFromString(s string, f *FunctionSelector) error {
	if utils.HasHexPrefix(s) {
		if !hexRegexp.Match([]byte(s)[2:]) {
			return fmt.Errorf("function selector %s must be 0x-hex encoded", s)
		}
		bytes := common.FromHex(s)
		if len(bytes) != FunctionSelectorLength {
			return errors.New("function ID must be 4 bytes in length")
		}
		f.SetBytes(bytes)
	} else {
		bytes, err := utils.Keccak256([]byte(s))
		if err != nil {
			return err
		}
		f.SetBytes(bytes[0:4])
	}
	return nil
}

func (f *FunctionSelector) UnmarshalJSON(input []byte) error {
	var s string
	err := json.Unmarshal(input, &s)
	if err != nil {
		return err
	}
	return unmarshalFromString(s, f)
}

func (f FunctionSelector) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f FunctionSelector) Value() (driver.Value, error) {
	return f.Bytes(), nil
}

func (f *FunctionSelector) Scan(value interface{}) error {
	temp, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unable to convent %v of type %T to FunctionSelector", value, value)
	}
	if len(temp) != FunctionSelectorLength {
		return fmt.Errorf("function selector %v should have length %d, but has length %d",
			temp, FunctionSelectorLength, len(temp))
	}
	copy(f[:], temp)
	return nil
}

type UntrustedBytes []byte

func (ary UntrustedBytes) SafeByteSlice(start int, end int) ([]byte, error) {
	if end > len(ary) || start > end || start < 0 || end < 0 {
		var empty []byte
		return empty, errors.New("out of bounds slice access")
	}
	return ary[start:end], nil
}
