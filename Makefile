BIN     := dotfiles
DIST    := dist
LDFLAGS := -s -w
GOFLAGS := -trimpath

.PHONY: help run local build build-mac build-mac-amd64 build-windows build-linux tidy fmt vet test clean

help: ## このヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

run: ## ビルドせずに管理画面を起動
	go run .

local: ## ホスト向けにビルドして ./$(BIN) を作る
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

build: build-mac build-windows ## mac (arm64) と windows (amd64) をクロスビルド

build-mac: ## darwin/arm64
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-darwin-arm64 .

build-mac-amd64: ## darwin/amd64
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-darwin-amd64 .

build-windows: ## windows/amd64
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN).exe .

build-linux: ## linux/amd64
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-linux-amd64 .

tidy: ## 依存を整理
	go mod tidy

fmt: ## gofmt
	go fmt ./...

vet: ## go vet
	go vet ./...

test: ## テスト
	go test ./...

clean: ## 生成物を削除
	rm -rf $(DIST) $(BIN) $(BIN).exe
