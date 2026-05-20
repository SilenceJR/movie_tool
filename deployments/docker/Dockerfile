FROM golang:1.26-bookworm AS builder

WORKDIR /src
COPY backend/go.mod ./backend/go.mod
COPY backend/go.sum ./backend/go.sum
COPY backend ./backend
WORKDIR /src/backend
RUN go build -o /out/movie-tool ./cmd/server

FROM debian:bookworm-slim
# FROM alpine:latest

WORKDIR /app
COPY --from=builder /out/movie-tool /app/movie-tool

ENV MOVIE_TOOL_PORT=8080
EXPOSE 8080

CMD ["/app/movie-tool"]
