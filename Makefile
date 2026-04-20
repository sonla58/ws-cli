.PHONY: build test install uninstall clean run release-snapshot

BIN     := ws
PREFIX  ?= $(HOME)/.local
BINDIR  := $(PREFIX)/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/ws

test:
	go test ./...

install: build
	@mkdir -p $(BINDIR)
	@install -m 0755 $(BIN) $(BINDIR)/$(BIN)
	@echo
	@echo "  ws installed to $(BINDIR)/$(BIN)"
	@echo
	@echo "  Next steps:"
	@echo "    1. Ensure $(BINDIR) is on your PATH:"
	@echo "         export PATH=\"$(BINDIR):\$$PATH\""
	@echo "    2. Add the shell integration (pick one):"
	@echo "         zsh:  echo 'eval \"\$$(ws init zsh)\"' >> ~/.zshrc"
	@echo "         bash: echo 'eval \"\$$(ws init bash)\"' >> ~/.bashrc"
	@echo "         fish: echo 'ws init fish | source' >> ~/.config/fish/config.fish"
	@echo "    3. Open a new terminal (or 'source' the rc file)"
	@echo "    4. Try it:  ws add"
	@echo

uninstall:
	@rm -f $(BINDIR)/$(BIN)
	@echo "  removed $(BINDIR)/$(BIN) (remove the 'eval \"\$$(ws init ...)\"' line from your rc file manually)"

clean:
	rm -f $(BIN)

run: build
	./$(BIN)

release-snapshot:
	goreleaser release --snapshot --clean
