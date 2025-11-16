# SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
# SPDX-License-Identifier: GPL-3.0-only

REVISION=$$(./.github/revision_get.sh)
GO_BUILD=go build -ldflags "-X main.buildInfo=revision:$(REVISION);" -trimpath
GO_INSTALL=go install -ldflags "-X main.buildInfo=revision:$(REVISION);" -trimpath

build: bz

bz: grpc
	$(GO_BUILD) -gcflags "all=-N -l"

install: bz
	$(GO_INSTALL)


test: .requirements
	go vet ./...
	deadcode ./... | grep -v "unreachable func" | tee .deadcode.out
	if [ -s .deadcode.out ]; then \
		echo "Dead code found" \
		rm -f .deadcode.out \
		exit 1; \
	fi
	nilaway -include-pkgs="github.com/bazurto/bz" ./...
	go vet -vettool $(shell which nilness) ./...
	$(GO_BUILD) -v ./...
	go test ./...

.requirements:
	go install golang.org/x/tools/cmd/deadcode@latest
	echo "go install golang.org/x/tools/cmd/deadcode@latest" >> .requirements
	go install go.uber.org/nilaway/cmd/nilaway@latest
	echo "go install go.uber.org/nilaway/cmd/nilaway@latest" >> .requirements
	go install golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness@latest
	echo "go install golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness@latest" >> .requirements

release: .revision.inc.txt bz-linux-amd64 bz-linux-arm64 bz-darwin-amd64 bz-darwin-arm64 bz-windows-amd64.exe
	gh release create --generate-notes -t v$(REVISION) v$(REVISION)
	gh release upload v$(REVISION) bz-linux-amd64
	gh release upload v$(REVISION) bz-linux-arm64
	gh release upload v$(REVISION) bz-darwin-amd64
	gh release upload v$(REVISION) bz-darwin-arm64
	gh release upload v$(REVISION) bz-windows-amd64.exe

dist: bz-linux-amd64 bz-linux-arm64 bz-darwin-amd64 bz-darwin-arm64 bz-windows-amd64.exe

bz-linux-amd64:
	GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o bz-linux-amd64
bz-linux-arm64:
	GOOS=linux   GOARCH=arm64 $(GO_BUILD) -o bz-linux-arm64
bz-darwin-amd64:
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o bz-darwin-amd64
bz-darwin-arm64:
	GOOS=darwin  GOARCH=arm64 $(GO_BUILD) -o bz-darwin-arm64
bz-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o bz-windows-amd64.exe

.revision.inc.txt:
	echo $$(./.github/revision_inc.sh) > .revision.inc.txt


grpc: grpc/bazurto/bazurto_grpc.pb.go grpc/bazurto/bazurto.pb.go
grpc/bazurto/bazurto_grpc.pb.go: bazurto.proto
	mkdir -p grpc/bazurto
	protoc --go_out=grpc/bazurto --go-grpc_out=grpc/bazurto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative bazurto.proto
grpc/bazurto/bazurto.pb.go: bazurto.proto
	mkdir -p grpc/bazurto
	protoc --go_out=grpc/bazurto --go-grpc_out=grpc/bazurto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative bazurto.proto


clean:
	rm -fr bz 
	rm -f bz-linux-amd64
	rm -f bz-linux-arm64
	rm -f bz-darwin-amd64
	rm -f bz-darwin-arm64
	rm -f bz-windows-amd64.exe
	rm -f .revision.inc.txt
	rm -f .requirements
	rm -f .deadcode.out
	rm -fr grpc


.PHONY: clean bz install dist grpc
