# Multi-stage Dockerfile für ein minimales Produktions-Image
# Verwende hier dieselbe/neuere Go-Version wie in go.mod (z.B. 1.23).
FROM golang:1.23 AS builder

WORKDIR /src

# Falls go.mod/go.sum vorhanden sind, werden sie zuerst kopiert zum schnelleren Caching
COPY go.mod go.sum ./
RUN if [ -f go.mod ]; then go mod download; fi

# Restlichen Quellcode kopieren und bauen
COPY . .

# Build-Flags
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