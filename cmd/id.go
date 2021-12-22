package cmd

type JAID struct {
	ID string `json:"id"`
}

func NewJAID(id string) JAID {
	return JAID{ID: id}
}

func (jaid JAID) GetID() string {
	return jaid.ID
}

func (jaid *JAID) SetID(value string) error {
	jaid.ID = value

	return nil
}
