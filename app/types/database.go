package types

import (
	"time"

	"gorm.io/gorm"
)

type GermanWord struct {
	gorm.Model

	DifficultyLevel string
	GermanWord      string
	Definition      string
	created_at      time.Time
	deleted_at      time.Time
}
