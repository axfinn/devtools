.PHONY: build-all clean

BINARY=proxy-client
DIST=dist

build-all: clean
	mkdir -p $(DIST)
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-darwin-arm64   .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-darwin-amd64   .
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-linux-amd64    .
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-windows-amd64.exe .
	@echo "构建完成:"
	@ls -lh $(DIST)/

clean:
	rm -rf $(DIST)

# 本地快速构建（当前平台）
build:
	go build -o $(BINARY) .
