package presenters

import (
	"strconv"
)

type JAID struct {
	ID string `json:"-"`
}

func NewJAID(id string) JAID {
	return JAID{id}
}

func NewJAIDInt32(id int32) JAID {
	return JAID{strconv.Itoa(int(id))}
}

func NewJAIDInt64(id int64) JAID {
	return JAID{strconv.Itoa(int(id))}
}

func NewJAIDUint(id uint) JAID {
	return JAID{strconv.Itoa(int(id))}
}

func (jaid JAID) GetID() string {
	return jaid.ID
}

func (jaid *JAID) SetID(value string) error {
	jaid.ID = value

	return nil
}
