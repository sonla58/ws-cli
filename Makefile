.PHONY: build test install clean run

BIN := ws

build:
	go build -o $(BIN) ./cmd/ws

test:
	go test ./...

install:
	go install ./cmd/ws

clean:
	rm -f $(BIN)

run: build
	./$(BIN)
