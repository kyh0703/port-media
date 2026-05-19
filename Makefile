TARGET := portfoilo-media
CMD := ./cmd/app

.PHONY: build run test tidy lint air

build:
	go build -o bin/$(TARGET) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

air:
	air
