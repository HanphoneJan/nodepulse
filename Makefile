BIN := nodepulse
VERSION ?= 0.3.0
LDFLAGS := -s -w -X github.com/HanphoneJan/nodepulse/internal/version.Version=$(VERSION)
UI_TAGS ?= production,webkit2_41
ADDR ?= 127.0.0.1:8099

.PHONY: build gui run web test race vet fmt icons ui-check deb appimage clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/nodepulse

gui:
	mkdir -p bin
	cd gui && go build -tags $(UI_TAGS) -ldflags "$(LDFLAGS)" -o ../bin/nodepulse-gui .

icons:
	go run ./internal/uiassets/genicons

ui-check:
	find internal/uiassets/dist -name '*.js' -not -path '*/material-color-utilities.js' -print0 | xargs -0 -n1 node --check
	go test ./internal/uiassets/...

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
