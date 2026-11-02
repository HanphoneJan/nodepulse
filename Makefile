BIN := nodepulse
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
UI_TAGS ?= webkit2_41
ADDR ?= 127.0.0.1:8099

.PHONY: build gui run web test race vet fmt deb appimage clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/nodepulse

gui:
	mkdir -p bin
	cd gui && go build -tags $(UI_TAGS) -o ../bin/nodepulse-gui .

run: gui
	./bin/nodepulse-gui

web: build
	./bin/nodepulse serve --addr $(ADDR)

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './gui/frontend/*')

deb: build gui
	mkdir -p dist
	VERSION=$(VERSION) nfpm package -f packaging/nfpm.yaml -p deb -t dist

appimage: build gui
	VERSION=$(VERSION) bash packaging/build-appimage.sh

clean:
	rm -rf bin dist
