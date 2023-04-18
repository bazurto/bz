# SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
# SPDX-License-Identifier: GPL-3.0-only

bz:
	go build -gcflags "-N -l"

build:
	go build -v ./...

install: bz
	go install -gcflags "-N -l"

test:
	go build -v ./...
 
release: release.tmp bz-linux-amd64 bz-linux-arm64 bz-darwin-amd64 bz-darwin-arm64 bz-windows-amd64.exe
	gh release create --generate-notes -t v$$(cat release.tmp) v$$(cat release.tmp)
	gh release upload v$$(cat release.tmp) bz-linux-amd64
	gh release upload v$$(cat release.tmp) bz-linux-arm64
	gh release upload v$$(cat release.tmp) bz-darwin-amd64
	gh release upload v$$(cat release.tmp) bz-darwin-arm64
	gh release upload v$$(cat release.tmp) bz-windows-amd64.exe

release.tmp:
	echo $$(./.github/version.sh) > release.tmp

dist: bz-linux-amd64 bz-linux-arm64 bz-darwin-amd64 bz-darwin-arm64 bz-windows-amd64.exe

bz-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bz-linux-amd64
bz-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bz-linux-arm64
bz-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o bz-darwin-amd64
bz-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bz-darwin-arm64
bz-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build -o bz-windows-amd64.exe

clean:
	rm -fr bz 
	rm -f bz-linux-amd64
	rm -f bz-linux-arm64
	rm -f bz-darwin-amd64
	rm -f bz-darwin-arm64
	rm -f bz-windows-amd64.exe
	rm -f release.tmp


.PHONY: clean bz install dist
