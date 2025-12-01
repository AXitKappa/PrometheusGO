# Multi-stage Dockerfile für ein minimales, zuverlässiges Produktions-Image
FROM golang:1.23 AS builder

WORKDIR /src

# Abhängigkeiten zuerst kopieren für Docker-Cache
COPY go.mod ./
ENV GOINSECURE="*"
ENV GOPROXY=direct
ENV GO111MODULE=on
RUN git config --global http.sslverify false
RUN go mod download || true

# Restlichen Quellcode kopieren und builden
COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
# Build in /bin damit klar ist, dass Binary an einem eigenen Ort landet
RUN go build -ldflags="-s -w" -o /bin/my-app .

# Laufzeit-Image
FROM alpine:3.18
# Erstelle non-root User
RUN addgroup -S app && adduser -S -G app app

# Binary an einen standard Ort kopieren
COPY --from=builder /bin/my-app /usr/local/bin/my-app
# Sicherstellen, dass das Binary ausführbar und für den app-User zugreifbar ist
RUN chown app:app /usr/local/bin/my-app && chmod 0755 /usr/local/bin/my-app

USER app
ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/my-app"]