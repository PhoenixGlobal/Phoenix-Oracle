package models

import (
	"crypto/subtle"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/util"
)

type BridgeTypeRequest struct {
	Name                   TaskType     `json:"name"`
	URL                    WebURL       `json:"url"`
	Confirmations          uint32       `json:"confirmations"`
	MinimumContractPayment *assets.Phb `json:"minimumContractPayment"`
}

func (bt BridgeTypeRequest) GetID() string {
	return bt.Name.String()
}

func (bt BridgeTypeRequest) GetName() string {
	return "bridges"
}

func (bt *BridgeTypeRequest) SetID(value string) error {
	name, err := NewTaskType(value)
	bt.Name = name
	return err
}

type BridgeTypeAuthentication struct {
	Name                   TaskType
	URL                    WebURL
	Confirmations          uint32
	IncomingToken          string
	OutgoingToken          string
	MinimumContractPayment *assets.Phb
}

type BridgeType struct {
	Name                   TaskType `gorm:"primary_key"`
	URL                    WebURL
	Confirmations          uint32
	IncomingTokenHash      string
	Salt                   string
	OutgoingToken          string
	MinimumContractPayment *assets.Phb `gorm:"type:varchar(255)"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func NewBridgeType(btr *BridgeTypeRequest) (*BridgeTypeAuthentication,
	*BridgeType, error) {
	incomingToken := utils.NewSecret(24)
	outgoingToken := utils.NewSecret(24)
	salt := utils.NewSecret(24)

	hash, err := incomingTokenHash(incomingToken, salt)
	if err != nil {
		return nil, nil, err
	}

	return &BridgeTypeAuthentication{
			Name:                   btr.Name,
			URL:                    btr.URL,
			Confirmations:          btr.Confirmations,
			IncomingToken:          incomingToken,
			OutgoingToken:          outgoingToken,
			MinimumContractPayment: btr.MinimumContractPayment,
		}, &BridgeType{
			Name:                   btr.Name,
			URL:                    btr.URL,
			Confirmations:          btr.Confirmations,
			IncomingTokenHash:      hash,
			Salt:                   salt,
			OutgoingToken:          outgoingToken,
			MinimumContractPayment: btr.MinimumContractPayment,
		}, nil
}

func AuthenticateBridgeType(bt *BridgeType, token string) (bool, error) {
	hash, err := incomingTokenHash(token, bt.Salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(bt.IncomingTokenHash)) == 1, nil
}

func incomingTokenHash(token, salt string) (string, error) {
	input := fmt.Sprintf("%s-%s", token, salt)
	hash, err := utils.Sha256(input)
	if err != nil {
		return "", err
	}
	return hash, nil
}

type BridgeMetaData struct {
	LatestAnswer *big.Int `json:"latestAnswer"`
	UpdatedAt    *big.Int `json:"updatedAt"` // A unix timestamp
}

type BridgeMetaDataJSON struct {
	Meta BridgeMetaData
}

func MarshalBridgeMetaData(latestAnswer *big.Int, updatedAt *big.Int) (map[string]interface{}, error) {
	b, err := json.Marshal(&BridgeMetaData{LatestAnswer: latestAnswer, UpdatedAt: updatedAt})
	if err != nil {
		return nil, err
	}
	var mp map[string]interface{}
	err = json.Unmarshal(b, &mp)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

type TaskType string

// NewTaskType returns a formatted Task type.
func NewTaskType(val string) (TaskType, error) {
	re := regexp.MustCompile("^[a-zA-Z0-9-_]*$")
	if !re.MatchString(val) {
		return TaskType(""), fmt.Errorf("task type validation: name %v contains invalid characters", val)
	}

	return TaskType(strings.ToLower(val)), nil
}

func MustNewTaskType(val string) TaskType {
	tt, err := NewTaskType(val)
	if err != nil {
		panic(fmt.Sprintf("%v is not a valid TaskType", val))
	}
	return tt
}

func (t *TaskType) UnmarshalJSON(input []byte) error {
	var aux string
	if err := json.Unmarshal(input, &aux); err != nil {
		return err
	}
	tt, err := NewTaskType(aux)
	*t = tt
	return err
}

// MarshalJSON converts a TaskType to a JSON byte slice.
func (t TaskType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t TaskType) String() string {
	return string(t)
}

func (t TaskType) Value() (driver.Value, error) {
	return string(t), nil
}

func (t *TaskType) Scan(value interface{}) error {
	temp, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to TaskType", value, value)
	}

	*t = TaskType(temp)
	return nil
}
