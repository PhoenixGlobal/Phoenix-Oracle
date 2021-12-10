package models

import (
	"crypto/subtle"
	"strings"
	"time"

	"PhoenixOracle/lib/auth"
	"PhoenixOracle/util"

	"github.com/pkg/errors"
)

type ExternalInitiatorRequest struct {
	Name string  `json:"name"`
	URL  *WebURL `json:"url,omitempty"`
}

type ExternalInitiator struct {
	ID             int64   `gorm:"primary_key"`
	Name           string  `gorm:"not null;unique"`
	URL            *WebURL `gorm:"url,omitempty"`
	AccessKey      string  `gorm:"not null"`
	Salt           string  `gorm:"not null"`
	HashedSecret   string  `gorm:"not null"`
	OutgoingSecret string  `gorm:"not null"`
	OutgoingToken  string  `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewExternalInitiator(
	eia *auth.Token,
	eir *ExternalInitiatorRequest,
) (*ExternalInitiator, error) {
	salt := utils.NewSecret(utils.DefaultSecretSize)
	hashedSecret, err := auth.HashedSecret(eia, salt)
	if err != nil {
		return nil, errors.Wrap(err, "error hashing secret for external initiator")
	}

	return &ExternalInitiator{
		Name:           strings.ToLower(eir.Name),
		URL:            eir.URL,
		AccessKey:      eia.AccessKey,
		HashedSecret:   hashedSecret,
		Salt:           salt,
		OutgoingToken:  utils.NewSecret(utils.DefaultSecretSize),
		OutgoingSecret: utils.NewSecret(utils.DefaultSecretSize),
	}, nil
}

func AuthenticateExternalInitiator(eia *auth.Token, ea *ExternalInitiator) (bool, error) {
	hashedSecret, err := auth.HashedSecret(eia, ea.Salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(hashedSecret), []byte(ea.HashedSecret)) == 1, nil
}
