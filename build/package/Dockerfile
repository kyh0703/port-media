FROM docker.io/library/golang:1.26-alpine AS build

# Build a static Linux binary.
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/portfoilo-media ./cmd/app

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=build /out/portfoilo-media /app/portfoilo-media
COPY dev.yaml ./dev.yaml

EXPOSE 8080

ENTRYPOINT ["/app/portfoilo-media"]
