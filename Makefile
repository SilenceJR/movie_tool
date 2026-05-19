.PHONY: backend-test backend-run backend-fmt docker-update install-git-hooks

backend-test:
	cd backend && GOCACHE=/private/tmp/movie_tool_go_cache go test ./...

backend-run:
	cd backend && GOCACHE=/private/tmp/movie_tool_go_cache go run ./cmd/server

backend-fmt:
	cd backend && gofmt -w ./cmd ./internal

docker-update:
	./scripts/docker-update.sh

install-git-hooks:
	./scripts/install-git-hooks.sh
