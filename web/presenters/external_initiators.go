package presenters

import (
	"fmt"
	"time"

	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/auth"
)

type ExternalInitiatorAuthentication struct {
	Name           string        `json:"name,omitempty"`
	URL            models.WebURL `json:"url,omitempty"`
	AccessKey      string        `json:"incomingAccessKey,omitempty"`
	Secret         string        `json:"incomingSecret,omitempty"`
	OutgoingToken  string        `json:"outgoingToken,omitempty"`
	OutgoingSecret string        `json:"outgoingSecret,omitempty"`
}

func NewExternalInitiatorAuthentication(
	ei models.ExternalInitiator,
	eia auth.Token,
) *ExternalInitiatorAuthentication {
	var result = &ExternalInitiatorAuthentication{
		Name:           ei.Name,
		AccessKey:      ei.AccessKey,
		Secret:         eia.Secret,
		OutgoingToken:  ei.OutgoingToken,
		OutgoingSecret: ei.OutgoingSecret,
	}
	if ei.URL != nil {
		result.URL = *ei.URL
	}
	return result
}

func (ei *ExternalInitiatorAuthentication) GetID() string {
	return ei.Name
}

func (*ExternalInitiatorAuthentication) GetName() string {
	return "external initiators"
}

func (ei *ExternalInitiatorAuthentication) SetID(name string) error {
	ei.Name = name
	return nil
}

type ExternalInitiatorResource struct {
	JAID
	Name          string         `json:"name"`
	URL           *models.WebURL `json:"url"`
	AccessKey     string         `json:"accessKey"`
	OutgoingToken string         `json:"outgoingToken"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

func NewExternalInitiatorResource(ei models.ExternalInitiator) ExternalInitiatorResource {
	return ExternalInitiatorResource{
		JAID:          NewJAID(fmt.Sprintf("%d", ei.ID)),
		Name:          ei.Name,
		URL:           ei.URL,
		AccessKey:     ei.AccessKey,
		OutgoingToken: ei.OutgoingToken,
		CreatedAt:     ei.CreatedAt,
		UpdatedAt:     ei.UpdatedAt,
	}
}

func (ExternalInitiatorResource) GetName() string {
	return "externalInitiators"
}
