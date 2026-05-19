package main

import (
	"context"
	"log"

	"movie-tool/backend/internal/ai"
	"movie-tool/backend/internal/api"
	"movie-tool/backend/internal/automation"
	"movie-tool/backend/internal/catalog"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/database"
	"movie-tool/backend/internal/integration"
	"movie-tool/backend/internal/library"
	"movie-tool/backend/internal/localization"
	"movie-tool/backend/internal/media"
	"movie-tool/backend/internal/organizer"
	"movie-tool/backend/internal/scraper"
	"movie-tool/backend/internal/strm"
	"movie-tool/backend/internal/task"
)

func main() {
	cfg := config.Load()

	db, err := database.OpenSQLite(context.Background(), database.SQLiteOptions{Path: cfg.Database})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := api.NewServerWithDependencies(cfg, api.Dependencies{
		AI:           ai.NewSQLStore(db),
		Automations:  automation.NewSQLStore(db),
		Catalog:      catalog.NewSQLStore(db),
		Integrations: integration.NewSQLStore(db),
		Libraries:    library.NewSQLStore(db),
		Localization: localization.NewSQLStore(db),
		MediaFiles:   media.NewSQLStore(db),
		Organizer:    organizer.NewSQLStore(db),
		Scraper:      scraper.NewSQLStore(db),
		STRM:         strm.NewSQLStore(db),
		Tasks:        task.NewQueueWithStore(task.NewSQLStore(db)),
	})

	log.Printf("movie-tool backend listening on %s", cfg.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
