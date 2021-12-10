package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"

	"PhoenixOracle/lib/logger"

	"PhoenixOracle/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// SignatureLength is the length of the signature in bytes: v = 1, r = 32, s
	// = 32; v + r + s = 65
	SignatureLength = 65
)

type Signature [SignatureLength]byte

func NewSignature(s string) (Signature, error) {
	bytes := common.FromHex(s)
	return BytesToSignature(bytes), nil
}

func BytesToSignature(b []byte) Signature {
	var s Signature
	s.SetBytes(b)
	return s
}

func (s Signature) Bytes() []byte { return s[:] }

func (s Signature) Big() *big.Int { return new(big.Int).SetBytes(s[:]) }

func (s Signature) Hex() string { return hexutil.Encode(s[:]) }

func (s Signature) String() string {
	return s.Hex()
}

func (s Signature) Format(state fmt.State, c rune) {
	_, err := fmt.Fprintf(state, "%"+string(c), s.String())
	logger.ErrorIf(err, "failed when format signature to state")
}

func (s *Signature) SetBytes(b []byte) {
	if len(b) > len(s) {
		b = b[len(b)-SignatureLength:]
	}

	copy(s[SignatureLength-len(b):], b)
}

func (s *Signature) UnmarshalText(input []byte) error {
	var err error
	*s, err = NewSignature(string(input))
	return err
}

func (s Signature) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Signature) UnmarshalJSON(input []byte) error {
	input = utils.RemoveQuotes(input)
	return s.UnmarshalText(input)
}

func (s Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s Signature) Value() (driver.Value, error) {
	return s.String(), nil
}

func (s *Signature) Scan(value interface{}) error {
	temp, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to Signature", value, value)
	}

	newSig, err := NewSignature(temp)
	*s = newSig
	return err
}
