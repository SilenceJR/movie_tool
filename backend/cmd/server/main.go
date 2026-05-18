package main

import (
	"context"
	"log"

	"movie-tool/backend/internal/api"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/database"
	"movie-tool/backend/internal/library"
	"movie-tool/backend/internal/media"
)

func main() {
	cfg := config.Load()

	db, err := database.OpenSQLite(context.Background(), database.SQLiteOptions{Path: cfg.Database})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := api.NewServerWithDependencies(cfg, api.Dependencies{
		Libraries:  library.NewSQLStore(db),
		MediaFiles: media.NewSQLStore(db),
	})

	log.Printf("movie-tool backend listening on %s", cfg.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
