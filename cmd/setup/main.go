package main

import (
	"log"

	"github.com/SkMarcin/Token-Transfer-API/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting API server setup...")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("ERROR: Could not connect to DB: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("ERROR: Migration failed: %v", err)
	}

	if err := database.SeedInitialBalance(db); err != nil {
		log.Fatalf("ERROR: Seeding failed: %v", err)
	}

	log.Println("Database setup complete.")
}
