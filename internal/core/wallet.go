package core

import "gorm.io/gorm"

type Wallet struct {
	gorm.Model
	Address string `gorm:"uniqueIndex;type:varchar(42);not null"`
	Balance int64  `gorm:"type:bigint;not null;default:0"`
}
