package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tidwall/gjson"
)

const (
	FormatBytes = "bytes"
	FormatPreformatted = "preformatted"
	FormatUint256 = "uint256"
	FormatInt256 = "int256"
	FormatBool = "bool"
)

func GenericEncode(types []string, values ...interface{}) ([]byte, error) {
	if len(values) != len(types) {
		return nil, errors.New("must include same number of values as types")
	}
	var args abi.Arguments
	for _, t := range types {
		ty, _ := abi.NewType(t, "", nil)
		args = append(args, abi.Argument{Type: ty})
	}
	out, err := args.PackValues(values)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func MustGenericEncode(types []string, values ...interface{}) []byte {
	out, err := GenericEncode(types, values)
	if err != nil {
		panic(err)
	}
	return out
}

func ConcatBytes(bufs ...[]byte) []byte {
	return bytes.Join(bufs, []byte{})
}

func EVMTranscodeBytes(value gjson.Result) ([]byte, error) {
	switch value.Type {
	case gjson.String:
		return EVMEncodeBytes([]byte(value.Str)), nil

	case gjson.False:
		return EVMEncodeBytes(EVMWordUint64(0)), nil

	case gjson.True:
		return EVMEncodeBytes(EVMWordUint64(1)), nil

	case gjson.Number:
		v := big.NewFloat(value.Num)
		vInt, _ := v.Int(nil)
		word, err := EVMWordSignedBigInt(vInt)
		if err != nil {
			return nil, errors.Wrap(err, "while converting float to int256")
		}
		return EVMEncodeBytes(word), nil
	default:
		return []byte{}, fmt.Errorf("unsupported encoding for value: %s", value.Type)
	}
}

func roundToEVMWordBorder(length int) int {
	mod := length % EVMWordByteLen
	if mod == 0 {
		return 0
	}
	return EVMWordByteLen - mod
}

func EVMEncodeBytes(input []byte) []byte {
	length := len(input)
	return ConcatBytes(
		EVMWordUint64(uint64(length)),
		input,
		make([]byte, roundToEVMWordBorder(length)))
}

func EVMTranscodeBool(value gjson.Result) ([]byte, error) {
	var output uint64

	switch value.Type {
	case gjson.Number:
		if value.Num != 0 {
			output = 1
		}

	case gjson.String:
		if len(value.Str) > 0 {
			output = 1
		}

	case gjson.True:
		output = 1

	case gjson.JSON:
		value.ForEach(func(key, value gjson.Result) bool {
			output = 1
			return false
		})

	case gjson.False, gjson.Null:

	default:
		panic(fmt.Errorf("unreachable/unsupported encoding for value: %s", value.Type))
	}

	return EVMWordUint64(output), nil
}

func parseDecimalString(input string) (*big.Int, error) {
	parseValue, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return nil, err
	}
	output, ok := big.NewInt(0).SetString(fmt.Sprintf("%.f", parseValue), 10)
	if !ok {
		return nil, fmt.Errorf("error parsing decimal %s", input)
	}
	return output, nil
}

func parseNumericString(input string) (*big.Int, error) {
	if HasHexPrefix(input) {
		output, ok := big.NewInt(0).SetString(RemoveHexPrefix(input), 16)
		if !ok {
			return nil, fmt.Errorf("error parsing hex %s", input)
		}
		return output, nil
	}

	output, ok := big.NewInt(0).SetString(input, 10)
	if !ok {
		return parseDecimalString(input)
	}
	return output, nil
}

func parseJSONAsEVMWord(value gjson.Result) (*big.Int, error) {
	output := new(big.Int)

	switch value.Type {
	case gjson.String:
		var err error
		output, err = parseNumericString(value.Str)
		if err != nil {
			return nil, err
		}

	case gjson.Number:
		output.SetInt64(int64(value.Num))

	case gjson.Null:

	default:
		return nil, fmt.Errorf("unsupported encoding for value: %s", value.Type)
	}

	return output, nil
}

func EVMTranscodeUint256(value gjson.Result) ([]byte, error) {
	output, err := parseJSONAsEVMWord(value)
	if err != nil {
		return nil, err
	}

	if output.Cmp(big.NewInt(0)) < 0 {
		return nil, fmt.Errorf("%v cannot be represented as uint256", output)
	}

	return EVMWordBigInt(output)
}

func EVMTranscodeInt256(value gjson.Result) ([]byte, error) {
	output, err := parseJSONAsEVMWord(value)
	if err != nil {
		return nil, err
	}

	return EVMWordSignedBigInt(output)
}

func EVMTranscodeJSONWithFormat(value gjson.Result, format string) ([]byte, error) {
	switch format {
	case FormatBytes:
		return EVMTranscodeBytes(value)
	case FormatPreformatted:
		return hex.DecodeString(RemoveHexPrefix(value.Str))
	case FormatUint256:
		data, err := EVMTranscodeUint256(value)
		if err != nil {
			return []byte{}, err
		}
		return EVMEncodeBytes(data), nil

	case FormatInt256:
		data, err := EVMTranscodeInt256(value)
		if err != nil {
			return []byte{}, err
		}
		return EVMEncodeBytes(data), nil

	case FormatBool:
		data, err := EVMTranscodeBool(value)
		if err != nil {
			return []byte{}, err
		}
		return EVMEncodeBytes(data), nil

	default:
		return []byte{}, fmt.Errorf("unsupported format: %s", format)
	}
}

func EVMWordAddress(val common.Address) []byte {
	word := make([]byte, EVMWordByteLen)
	start := EVMWordByteLen - 20
	for i, b := range val.Bytes() {
		word[start+i] = b
	}
	return word
}

func EVMWordUint64(val uint64) []byte {
	word := make([]byte, EVMWordByteLen)
	binary.BigEndian.PutUint64(word[EVMWordByteLen-8:], val)
	return word
}

func EVMWordUint32(val uint32) []byte {
	word := make([]byte, EVMWordByteLen)
	binary.BigEndian.PutUint32(word[EVMWordByteLen-4:], val)
	return word
}

func EVMWordUint128(val *big.Int) ([]byte, error) {
	bytes := val.Bytes()
	if val.BitLen() > 128 {
		return nil, fmt.Errorf("overflow saving uint128 to EVM word: %v", val)
	} else if val.Sign() == -1 {
		return nil, fmt.Errorf("invalid attempt to save negative value as uint128 to EVM word: %v", val)
	}
	return common.LeftPadBytes(bytes, EVMWordByteLen), nil
}

func EVMWordSignedBigInt(val *big.Int) ([]byte, error) {
	bytes := val.Bytes()
	if val.BitLen() > (8*EVMWordByteLen - 1) {
		return nil, fmt.Errorf("overflow saving signed big.Int to EVM word: %v", val)
	}
	if val.Sign() == -1 {
		twosComplement := new(big.Int).Add(val, MaxUint256)
		bytes = new(big.Int).Add(twosComplement, big.NewInt(1)).Bytes()
	}
	return common.LeftPadBytes(bytes, EVMWordByteLen), nil
}

func EVMWordBigInt(val *big.Int) ([]byte, error) {
	if val.Sign() == -1 {
		return nil, errors.New("Uint256 cannot be negative")
	}
	bytes := val.Bytes()
	if len(bytes) > EVMWordByteLen {
		return nil, fmt.Errorf("overflow saving big.Int to EVM word: %v", val)
	}
	return common.LeftPadBytes(bytes, EVMWordByteLen), nil
}

func Bytes32FromString(s string) [32]byte {
	var b32 [32]byte
	copy(b32[:], s)
	return b32
}

func Bytes4FromString(s string) [4]byte {
	var b4 [4]byte
	copy(b4[:], s)
	return b4
}

var (
	maxUint257 = &big.Int{}
	MaxUint256 = &big.Int{}
	MaxInt256 = &big.Int{}
	MinInt256 = &big.Int{}
)

func init() {
	maxUint257 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	MaxUint256 = new(big.Int).Sub(maxUint257, big.NewInt(1))
	MaxInt256 = new(big.Int).Div(MaxUint256, big.NewInt(2))
	MinInt256 = new(big.Int).Neg(MaxInt256)
}
