BUILD_OUTPUT=build
ASSET_PATH=assets

CUSTOM=-X 'main.buildDate=$(shell date)' -X 'main.gitHash=$(shell git rev-parse --short HEAD)' -X 'main.buildOn=$(shell go version)'
LDFLAGS=$(CUSTOM) -w -s -extldflags=-static
GO_BUILD=go build -trimpath -ldflags "$(LDFLAGS)"

APP_NAMES=fileserver

FILESERVER_PATH=./cmd/fileserver
FILESERVER_BIN_NAME=fileserver

PM_PATH=./cmd/pm
PM_BIN_NAME=pm

define GO_BUILD_APP
	CGO_ENABLED=1 GOOS=$(1) GOARCH=$(2) $(GO_BUILD) -o $(BUILD_OUTPUT)/$(3) $(4)
endef

.PHONY: all
all: fileserver

.PHONY: fmt
fmt:
	gofumpt -l -w -extra .

.PHONY: tidy
tidy:
	@echo "[main] tidy"
	go mod tidy

.PHONY: update
update:
	@echo "[main] update dependencies"
	go get -u ./...

.PHONY: lint
lint: fmt
	@echo "[main] golangci-lint"
	golangci-lint run ./... --fix

.PHONY: test
test:
	go test ./...

.PHONY: deadcode
deadcode:
	deadcode ./...

.PHONY: syso
syso:
	windres $(FILESERVER_PATH)/app.rc -O coff -o $(FILESERVER_PATH)/app.syso

.PHONY: png-to-icos
png-to-icos:
	magick $(ASSET_PATH)/win-icon.png -background none -define icon:auto-resize=256,128,64,48,32,16 $(ASSET_PATH)/win-icon.ico

.PHONY: copy-assets
copy-assets:
	cp -r $(ASSET_PATH)/* $(BUILD_OUTPUT)

.PHONY: gen-certs
gen-certs:
	mkcert localhost 127.0.0.1 ::1

# ----- FileServer -----
.PHONY: fileserver
fileserver: fileserver-linux fileserver-linux-arm64 fileserver-darwin fileserver-darwin-arm64 fileserver-windows

.PHONY: fileserver-linux
fileserver-linux: fmt
	$(call GO_BUILD_APP,linux,amd64,$(FILESERVER_BIN_NAME)-linux,$(FILESERVER_PATH))

.PHONY: fileserver-linux-arm64
fileserver-linux-arm64: fmt
	$(call GO_BUILD_APP,linux,arm64,$(FILESERVER_BIN_NAME)-linux-arm64,$(FILESERVER_PATH))

.PHONY: fileserver-darwin
fileserver-darwin: fmt
	$(call GO_BUILD_APP,darwin,amd64,$(FILESERVER_BIN_NAME)-darwin,$(FILESERVER_PATH))

.PHONY: fileserver-darwin-arm64
fileserver-darwin-arm64: fmt
	$(call GO_BUILD_APP,darwin,arm64,$(FILESERVER_BIN_NAME)-darwin-arm64,$(FILESERVER_PATH))

.PHONY: fileserver-windows
fileserver-windows: fmt copy-assets
	$(call GO_BUILD_APP,windows,amd64,$(FILESERVER_BIN_NAME).exe,$(FILESERVER_PATH))

# ----- PM -----
.PHONY: pm
pm: pm-linux pm-linux-arm64 pm-darwin pm-darwin-arm64 pm-windows

.PHONY: pm-linux
pm-linux: fmt
	$(call GO_BUILD_APP,linux,amd64,$(PM_BIN_NAME)-linux,$(PM_PATH))

.PHONY: pm-linux-arm64
pm-linux-arm64: fmt
	$(call GO_BUILD_APP,linux,arm64,$(PM_BIN_NAME)-linux-arm64,$(PM_PATH))

.PHONY: pm-darwin
pm-darwin: fmt
	$(call GO_BUILD_APP,darwin,amd64,$(PM_BIN_NAME)-darwin,$(PM_PATH))

.PHONY: pm-darwin-arm64
pm-darwin-arm64: fmt
	$(call GO_BUILD_APP,darwin,arm64,$(PM_BIN_NAME)-darwin-arm64,$(PM_PATH))

.PHONY: pm-windows
pm-windows: fmt copy-assets
	$(call GO_BUILD_APP,windows,amd64,$(PM_BIN_NAME).exe,$(PM_PATH))
