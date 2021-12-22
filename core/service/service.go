package service

import "PhoenixOracle/lib/health"

type (
	Service interface {
		Start() error
		Close() error

		health.Checkable
	}
)
