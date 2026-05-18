FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY backend/go.mod ./backend/go.mod
COPY backend ./backend
WORKDIR /src/backend
RUN go build -o /out/movie-tool ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/movie-tool /app/movie-tool

ENV MOVIE_TOOL_PORT=8080
EXPOSE 8080

CMD ["/app/movie-tool"]

