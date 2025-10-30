package database

import (
	"fmt"
	"log"

	"github.com/SkMarcin/Token-Transfer-API/internal/core"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("Running GORM migrations...")

	err := db.AutoMigrate(&core.Wallet{})
	if err != nil {
		return fmt.Errorf("failed to auto migrate Wallet model: %w", err)
	}
	log.Println("Migrations complete.")

	return nil
}
