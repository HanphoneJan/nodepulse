BIN := nodepulse
VERSION ?= 0.3.2
LDFLAGS := -s -w -X github.com/HanphoneJan/nodepulse/internal/version.Version=$(VERSION)
FLUTTER ?= flutter
DESKTOP := desktop
FLUTTER_BUNDLE := $(DESKTOP)/build/linux/x64/release/bundle
ADDR ?= 127.0.0.1:8099

.PHONY: build desktop android run test race vet fmt analyze deb appimage clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/nodepulse

desktop: build
	cd $(DESKTOP) && $(FLUTTER) build linux --release
	cp bin/$(BIN) $(FLUTTER_BUNDLE)/nodepulse-core

run: build
	cd $(DESKTOP) && NODEPULSE_BIN=$(CURDIR)/bin/$(BIN) $(FLUTTER) run -d linux

test:
	go test ./...
	cd $(DESKTOP) && $(FLUTTER) test

race:
	go test -race ./...

vet:
	go vet ./...

analyze:
	cd $(DESKTOP) && $(FLUTTER) analyze

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
	cd $(DESKTOP) && dart format lib test

deb: desktop
	mkdir -p dist
	VERSION=$(VERSION) nfpm package -f packaging/nfpm.yaml -p deb -t dist/NodePulse-linux-amd64-$(VERSION).deb

appimage: desktop
	VERSION=$(VERSION) bash packaging/build-appimage.sh

android: build
	VERSION=$(VERSION) bash packaging/build-android.sh

clean:
	rm -rf bin dist $(DESKTOP)/build
