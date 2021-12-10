package logger

import (
	"time"
)

type LogConfig struct {
	ID          int64  `gorm:"primary_key"`
	ServiceName string `gorm:"not null"`
	LogLevel    string `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
