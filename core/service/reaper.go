package service

import (
	"time"

	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
	"gorm.io/gorm"
)

type sessionReaper struct {
	db     *gorm.DB
	config SessionReaperConfig
}

type SessionReaperConfig interface {
	SessionTimeout() models.Duration
	ReaperExpiration() models.Duration
}

func NewSessionReaper(db *gorm.DB, config SessionReaperConfig) utils.SleeperTask {
	return utils.NewSleeperTask(&sessionReaper{
		db,
		config,
	})
}

func (sr *sessionReaper) Work() {
	recordCreationStaleThreshold := sr.config.ReaperExpiration().Before(
		sr.config.SessionTimeout().Before(time.Now()))
	err := sr.deleteStaleSessions(recordCreationStaleThreshold)
	if err != nil {
		logger.Error("unable to reap stale sessions: ", err)
	}
}

func (sr *sessionReaper) deleteStaleSessions(before time.Time) error {
	return sr.db.Exec("DELETE FROM sessions WHERE last_used < ?", before).Error
}
