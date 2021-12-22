package presenters

import "PhoenixOracle/lib/health"

type Check struct {
	JAID
	Name   string        `json:"name"`
	Status health.Status `json:"status"`
	Output string        `json:"output"`
}

func (c Check) GetName() string {
	return "checks"
}
