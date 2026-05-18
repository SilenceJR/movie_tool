.PHONY: backend-test backend-run backend-fmt

backend-test:
	cd backend && GOCACHE=/private/tmp/movie_tool_go_cache go test ./...

backend-run:
	cd backend && GOCACHE=/private/tmp/movie_tool_go_cache go run ./cmd/server

backend-fmt:
	cd backend && gofmt -w ./cmd ./internal

