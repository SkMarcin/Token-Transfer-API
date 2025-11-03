package database

import (
	"log"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var testDB *gorm.DB

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
