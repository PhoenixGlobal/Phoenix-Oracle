package auth

import (
	"encoding/hex"
	"fmt"

	"PhoenixOracle/util"

	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

var (
	ErrorAuthFailed = errors.New("Authentication failed")
)

type Token struct {
	AccessKey string `json:"accessKey"`
	Secret    string `json:"secret"`
}

func (ta *Token) GetID() string {
	return ta.AccessKey
}

func (ta *Token) GetName() string {
	return "auth_tokens"
}

func (ta *Token) SetID(id string) error {
	ta.AccessKey = id
	return nil
}

func NewToken() *Token {
	return &Token{
		AccessKey: utils.NewBytes32ID(),
		Secret:    utils.NewSecret(utils.DefaultSecretSize),
	}
}

func hashInput(ta *Token, salt string) []byte {
	return []byte(fmt.Sprintf("v0-%s-%s-%s", ta.AccessKey, ta.Secret, salt))
}

func HashedSecret(ta *Token, salt string) (string, error) {
	hasher := sha3.New256()
	_, err := hasher.Write(hashInput(ta, salt))
	if err != nil {
		return "", errors.Wrap(err, "error writing external initiator authentication to hasher")
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
