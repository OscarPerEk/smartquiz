package types

import (
	"gorm.io/gorm"
)

type GermanWord struct {
	gorm.Model // includes id, created_at, updated_at, deleted_at

	Example     string `gorm:"column:example;unique;not null"`
	GermanWord  string `gorm:"column:german_word;unique;not null"`
	Definition  string `gorm:"column:definition;unique;not null"`
	Translation string `gorm:"column:translation;unique;not null"`
}
