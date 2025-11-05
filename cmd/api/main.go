package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/SkMarcin/Token-Transfer-API/graph"
	"github.com/SkMarcin/Token-Transfer-API/internal/core"
	"github.com/SkMarcin/Token-Transfer-API/internal/database"
	"github.com/joho/godotenv"
)

const defaultPort = "8080"

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
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

	walletService := core.NewWalletService(db)

	srv := handler.New(graph.NewExecutableSchema(
		graph.Config{
			Resolvers: &graph.Resolver{
				WalletService: walletService,
			},
		},
	))

	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
