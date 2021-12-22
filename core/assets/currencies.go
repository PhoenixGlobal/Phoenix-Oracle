package assets

import (
	"database/sql/driver"
	"fmt"
	"math/big"

	"PhoenixOracle/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

var ErrNoQuotesForCurrency = errors.New("cannot unmarshal json.Number into currency")

// returns 10**precision.
func getDenominator(precision int) *big.Int {
	x := big.NewInt(10)
	return new(big.Int).Exp(x, big.NewInt(int64(precision)), nil)
}

func format(i *big.Int, precision int) string {
	r := big.NewRat(1, 1).SetFrac(i, getDenominator(precision))
	return fmt.Sprintf("%v", r.FloatString(precision))
}

// represent the smallest units of PHB
type Phb big.Int

func NewPhb(w int64) *Phb {
	return (*Phb)(big.NewInt(w))
}

func (l *Phb) String() string {
	if l == nil {
		return "0"
	}
	return fmt.Sprintf("%v", (*big.Int)(l))
}

func (l *Phb) Phb() string {
	if l == nil {
		return "0"
	}
	return format((*big.Int)(l), 18)
}

func (l *Phb) SetInt64(w int64) *Phb {
	return (*Phb)((*big.Int)(l).SetInt64(w))
}

func (l *Phb) ToInt() *big.Int {
	return (*big.Int)(l)
}

func (l *Phb) ToHash() common.Hash {
	return common.BigToHash((*big.Int)(l))
}

func (l *Phb) Set(x *Phb) *Phb {
	il := (*big.Int)(l)
	ix := (*big.Int)(x)

	w := il.Set(ix)
	return (*Phb)(w)
}

func (l *Phb) SetString(s string, base int) (*Phb, bool) {
	w, ok := (*big.Int)(l).SetString(s, base)
	return (*Phb)(w), ok
}

func (l *Phb) Cmp(y *Phb) int {
	return (*big.Int)(l).Cmp((*big.Int)(y))
}

func (l *Phb) Add(x, y *Phb) *Phb {
	il := (*big.Int)(l)
	ix := (*big.Int)(x)
	iy := (*big.Int)(y)

	return (*Phb)(il.Add(ix, iy))
}

func (l *Phb) Text(base int) string {
	return (*big.Int)(l).Text(base)
}

func (l *Phb) MarshalText() ([]byte, error) {
	return (*big.Int)(l).MarshalText()
}

func (l Phb) MarshalJSON() ([]byte, error) {
	value, err := l.MarshalText()
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(`"%s"`, value)), nil
}

func (l *Phb) UnmarshalJSON(data []byte) error {
	if utils.IsQuoted(data) {
		return l.UnmarshalText(utils.RemoveQuotes(data))
	}
	return ErrNoQuotesForCurrency
}

func (l *Phb) UnmarshalText(text []byte) error {
	if _, ok := l.SetString(string(text), 10); !ok {
		return fmt.Errorf("assets: cannot unmarshal %q into a *assets.Phb", text)
	}
	return nil
}

func (l *Phb) IsZero() bool {
	zero := big.NewInt(0)
	return (*big.Int)(l).Cmp(zero) == 0
}

func (*Phb) Symbol() string {
	return "PHB"
}

func (l Phb) Value() (driver.Value, error) {
	b := (big.Int)(l)
	return b.String(), nil
}

func (l *Phb) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		decoded, ok := l.SetString(v, 10)
		if !ok {
			return fmt.Errorf("unable to set string %v of %T to base 10 big.Int for Phb", value, value)
		}
		*l = *decoded
	case []uint8:
		decoded, ok := l.SetString(string(v), 10)
		if !ok {
			return fmt.Errorf("unable to set string %v of %T to base 10 big.Int for Phb", value, value)
		}
		*l = *decoded
	case int64:
		return fmt.Errorf("unable to convert %v of %T to Phb, is the sql type set to varchar?", value, value)
	default:
		return fmt.Errorf("unable to convert %v of %T to Phb", value, value)
	}

	return nil
}

type Eth big.Int

func NewEth(w int64) *Eth {
	return (*Eth)(big.NewInt(w))
}

func NewEthValue(w int64) Eth {
	eth := NewEth(w)
	return *eth
}

func NewEthValueS(s string) (Eth, error) {
	e, err := decimal.NewFromString(s)
	if err != nil {
		return Eth{}, err
	}
	w := e.Mul(decimal.RequireFromString("10").Pow(decimal.RequireFromString("18")))
	return *(*Eth)(w.BigInt()), nil
}

func (e *Eth) Cmp(y *Eth) int {
	return e.ToInt().Cmp(y.ToInt())
}

func (e *Eth) String() string {
	return format(e.ToInt(), 18)
}

func (e *Eth) SetInt64(w int64) *Eth {
	return (*Eth)(e.ToInt().SetInt64(w))
}

func (e *Eth) SetString(s string, base int) (*Eth, bool) {
	w, ok := e.ToInt().SetString(s, base)
	return (*Eth)(w), ok
}

func (e Eth) MarshalJSON() ([]byte, error) {
	value, err := e.MarshalText()
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(`"%s"`, value)), nil
}

func (e *Eth) MarshalText() ([]byte, error) {
	return e.ToInt().MarshalText()
}

func (e *Eth) UnmarshalJSON(data []byte) error {
	if utils.IsQuoted(data) {
		return e.UnmarshalText(utils.RemoveQuotes(data))
	}
	return ErrNoQuotesForCurrency
}

func (e *Eth) UnmarshalText(text []byte) error {
	if _, ok := e.SetString(string(text), 10); !ok {
		return fmt.Errorf("assets: cannot unmarshal %q into a *assets.Eth", text)
	}
	return nil
}

func (e *Eth) IsZero() bool {
	zero := big.NewInt(0)
	return e.ToInt().Cmp(zero) == 0
}

func (*Eth) Symbol() string {
	return "ETH"
}

func (e *Eth) ToInt() *big.Int {
	return (*big.Int)(e)
}

func (e *Eth) Scan(value interface{}) error {
	return (*utils.Big)(e).Scan(value)
}

func (e Eth) Value() (driver.Value, error) {
	return (utils.Big)(e).Value()
}
