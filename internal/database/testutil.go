package database

import (
	"fmt"
	"log"
	"testing"

	"github.com/SkMarcin/Token-Transfer-API/internal/core"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func SeedTestWallets(DB *gorm.DB) error {
	walletsToSeed := []core.Wallet{
		{
			Address: "0x0000000000000000000000000000000000000000",
			Balance: 10,
		},
		{
			Address: "0x2222222222222222222222222222222222222222",
			Balance: 10,
		},
	}

	for i, wallet := range walletsToSeed {
		result := DB.Where("address = ?", wallet.Address).Assign("balance", wallet.Balance).FirstOrCreate(&wallet)

		if result.Error != nil {
			return fmt.Errorf("failed to seed wallet %d (%s): %w", i+1, wallet.Address, result.Error)
		}

		if result.RowsAffected > 0 {
			log.Printf("Seeded wallet %d: %s with balance %d", i+1, wallet.Address, wallet.Balance)
		} else {
			log.Printf("Wallet %d (%s) already exists, balance reset to %d.", i+1, wallet.Address, wallet.Balance)
		}
	}

	return nil
}

func SetupTestDB(t *testing.T) *gorm.DB {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if testDB == nil {
		var err error

		testDB, err = Connect()
		if err != nil {
			t.Fatalf("Failed to connect to test database: %v", err)
		}

		if err := RunMigrations(testDB); err != nil {
			t.Fatalf("Failed to run migrations on test DB: %v", err)
		}
	}

	if err := SeedInitialBalance(testDB); err != nil {
		t.Fatalf("Failed to seed initial data for test DB: %v", err)
	}

	return testDB
}

func ClearTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE wallets RESTART IDENTITY CASCADE;")
}
