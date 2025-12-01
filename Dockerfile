# Multi-stage Dockerfile für ein minimales Produktions-Image
# Annahme: das main package liegt im Repo-Root und kann mit `go build -o /app .` gebaut werden.
FROM golang:1.20-alpine AS builder

WORKDIR /src

# Falls go.mod/go.sum vorhanden sind, werden sie zuerst kopiert zum schnelleren Caching
COPY go.mod go.sum ./
RUN if [ -f go.mod ]; then go mod download; fi

# Restlichen Quellcode kopieren und bauen
COPY . .

# Build-Flags: statisch(er), Stripping für kleinere Binaries
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /app .

# Minimales Laufzeit-Image
FROM alpine:3.18
RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /app /app
USER app
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app"]