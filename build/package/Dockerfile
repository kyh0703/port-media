FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/portfoilo-media ./cmd/app

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/portfoilo-media /app/portfoilo-media
COPY dev.yaml /app/dev.yaml
EXPOSE 8080
ENTRYPOINT ["/app/portfoilo-media"]
