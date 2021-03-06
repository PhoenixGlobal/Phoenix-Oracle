package ethkey

import "time"

type State struct {
	ID        int32 `gorm:"primary_key"`
	Address   EIP55Address
	NextNonce int64
	IsFunding bool
	CreatedAt time.Time
	UpdatedAt time.Time
	lastUsed  time.Time
}

func (State) TableName() string {
	return "eth_key_states"
}

func (s State) KeyID() string {
	return s.Address.Hex()
}

func (s State) LastUsed() time.Time {
	return s.lastUsed
}

func (s *State) WasUsed() {
	s.lastUsed = time.Now()
}
