package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectSuccess(t *testing.T) {
	a := assert.New(t)

	db, err := Connect()
	a.NoError(err, "Connect should not return error when db is running")

	if err != nil {
		t.Skipf("Connection failed: %v", err)
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying *sql.DB from GORM: %v", err)
	}
	defer sqlDB.Close()

	err = sqlDB.Ping()
	a.NoError(err, "The database connection should respond to ping")
}

func TestConnectFailure(t *testing.T) {
	a := assert.New(t)

	port := os.Getenv("DB_PORT")
	os.Setenv("DB_PORT", "12345")
	defer os.Setenv("DB_PORT", port)

	db, err := Connect()
	a.Error(err, "Connect() should return an error when connecting to a bad port")

	// Close connection if it succeeded
	if err == nil && db != nil {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}
