# SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
# SPDX-License-Identifier: GPL-3.0-only


REVISION=$$(./.github/revision_get.sh)

GO_BUILD=go build -ldflags "-X main.buildInfo=revision:$(REVISION);"

bz:
	$(GO_BUILD) -gcflags "-N -l"

build:
	go build -v ./...

install: bz
	go install -gcflags "-N -l"

test:
	go build -v ./...
 
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


clean:
	rm -fr bz 
	rm -f bz-linux-amd64
	rm -f bz-linux-arm64
	rm -f bz-darwin-amd64
	rm -f bz-darwin-arm64
	rm -f bz-windows-amd64.exe
	rm -f .revision.inc.txt


.PHONY: clean bz install dist
