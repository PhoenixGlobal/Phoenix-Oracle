package types

import (
	"context"

	"PhoenixOracle/core/service"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
)

type Tracker interface {
	HighestSeenHeadFromDB() (*models.Head, error)
	Start() error
	Stop() error
	SetLogger(logger *logger.Logger)
	Ready() error
	Healthy() error
}

type HeadTrackable interface {
	OnNewLongestChain(ctx context.Context, head models.Head)
}

type HeadBroadcasterRegistry interface {
	Subscribe(callback HeadTrackable) (currentLongestChain *models.Head, unsubscribe func())
}

type HeadBroadcaster interface {
	service.Service
	HeadTrackable
	Subscribe(callback HeadTrackable) (currentLongestChain *models.Head, unsubscribe func())
}
