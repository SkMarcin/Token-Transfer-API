package database

import (
	"fmt"
	"log"

	"github.com/SkMarcin/Token-Transfer-API/internal/models"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("Running GORM migrations...")

	err := db.AutoMigrate(&models.Wallet{})
	if err != nil {
		return fmt.Errorf("failed to auto migrate Wallet model: %w", err)
	}
	log.Println("Migrations complete.")

	return nil
}

func SeedInitialBalance(db *gorm.DB) error {
	log.Println("Seeding initial wallet balance...")

	initialWallet := models.Wallet{
		Address: "0x0000000000000000000000000000000000000000",
		Balance: 1000000,
	}

	result := db.FirstOrCreate(&initialWallet, models.Wallet{Address: initialWallet.Address})

	if result.Error != nil {
		return fmt.Errorf("failed to seed initial wallet: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("Seeded initial wallet: %s with balance %d", initialWallet.Address, initialWallet.Balance)
	} else {
		log.Println("Initial wallet already exists, skipping seed.")
	}

	return nil
}
