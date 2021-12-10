package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const base10 = 10

type BigFloat big.Float

func (b BigFloat) MarshalJSON() ([]byte, error) {
	var j = big.Float(b)
	return json.Marshal(&j)
}

func (b *BigFloat) UnmarshalJSON(buf []byte) error {
	var f float64
	if err := json.Unmarshal(buf, &f); err == nil {
		*b = BigFloat(*big.NewFloat(f))
		return nil
	}
	var bf big.Float
	if err := json.Unmarshal(buf, &bf); err != nil {
		return err
	}
	*b = BigFloat(bf)
	return nil
}

func (b *BigFloat) Value() *big.Float {
	return (*big.Float)(b)
}

type Big big.Int

func NewBig(i *big.Int) *Big {
	if i != nil {
		b := Big(*i)
		return &b
	}
	return nil
}

func NewBigI(i int64) *Big {
	return NewBig(big.NewInt(i))
}

func (b Big) MarshalText() ([]byte, error) {
	return []byte((*big.Int)(&b).Text(base10)), nil
}

func (b Big) MarshalJSON() ([]byte, error) {
	text, err := b.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (b *Big) UnmarshalText(input []byte) error {
	input = RemoveQuotes(input)
	str := string(input)
	if HasHexPrefix(str) {
		decoded, err := hexutil.DecodeBig(str)
		if err != nil {
			return err
		}
		*b = Big(*decoded)
		return nil
	}

	_, ok := b.setString(str, 10)
	if !ok {
		return fmt.Errorf("unable to convert %s to Big", str)
	}
	return nil
}

func (b *Big) setString(s string, base int) (*Big, bool) {
	w, ok := (*big.Int)(b).SetString(s, base)
	return (*Big)(w), ok
}

func (b *Big) UnmarshalJSON(input []byte) error {
	return b.UnmarshalText(input)
}

func (b Big) Value() (driver.Value, error) {
	return b.String(), nil
}

func (b *Big) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		decoded, ok := b.setString(v, 10)
		if !ok {
			return fmt.Errorf("unable to set string %v of %T to base 10 big.Int for Big", value, value)
		}
		*b = *decoded
	case []uint8:
		decoded, ok := b.setString(string(v), 10)
		if !ok {
			return fmt.Errorf("unable to set string %v of %T to base 10 big.Int for Big", value, value)
		}
		*b = *decoded
	default:
		return fmt.Errorf("unable to convert %v of %T to Big", value, value)
	}

	return nil
}

func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

func (b *Big) String() string {
	return b.ToInt().Text(10)
}

func (b *Big) Hex() string {
	return hexutil.EncodeBig(b.ToInt())
}

type BigIntSlice []*big.Int

func (s BigIntSlice) Len() int           { return len(s) }
func (s BigIntSlice) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s BigIntSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s BigIntSlice) Sort() {
	sort.Sort(s)
}

func (s BigIntSlice) Max() *big.Int {
	tmp := make(BigIntSlice, len(s))
	copy(tmp, s)
	tmp.Sort()
	return tmp[len(tmp)-1]
}

func (s BigIntSlice) Min() *big.Int {
	tmp := make(BigIntSlice, len(s))
	copy(tmp, s)
	tmp.Sort()
	return tmp[0]
}
