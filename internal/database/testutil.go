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

func SeedSecondWalletTest() error {
	log.Println("Seeding second wallet balance...")

	secondWallet := core.Wallet{
		Address: "0x2222222222222222222222222222222222222222",
		Balance: 1000000,
	}

	result := testDB.FirstOrCreate(&secondWallet, core.Wallet{Address: secondWallet.Address})

	if result.Error != nil {
		return fmt.Errorf("failed to seed second wallet: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("Seeded second wallet: %s with balance %d", secondWallet.Address, secondWallet.Balance)
	} else {
		log.Println("Second wallet already exists, skipping seed.")
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
