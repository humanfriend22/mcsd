PI ?= pi@raspberrypi.local

web-build:
	cd web && bun run generate

build:
	GOOS=linux GOARCH=arm64 go build -tags embed_web -ldflags="-s -w" -o mcsd .

deploy: web-build build
	rsync -az --progress -e "ssh -c aes128-ctr" mcsd $(PI):/tmp/mcsd
	ssh -t $(PI) "sudo mv /tmp/mcsd /usr/local/bin/mcsd && sudo chmod +x /usr/local/bin/mcsd"

clean:
	rm -f mcsd

.PHONY: build web-build deploy clean
