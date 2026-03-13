APP := hawkings
BIN_DIR := bin
DIST_DIR := dist

.PHONY: build test release clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP) ./cmd/$(APP)

test:
	go test ./...

release:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/$(APP)-darwin-arm64 ./cmd/$(APP)
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/$(APP)-linux-amd64 ./cmd/$(APP)
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(APP)-windows-amd64.exe ./cmd/$(APP)

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
