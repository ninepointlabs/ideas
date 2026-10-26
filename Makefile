BINARY  := idea
PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin
VERSION ?= $(or $(patsubst v%,%,$(shell git describe --tags --always --dirty 2>/dev/null)),dev)

.PHONY: build install uninstall clean

build:
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
