.PHONY: build build-cmd

all: build

build:
	docker run \
		--rm \
		-v "$(PWD)":/usr/src/myapp \
		-w /usr/src/myapp \
		--user 1000:1000 \
		-e XDG_CACHE_HOME=/tmp/.cache \
		golang:1.19 \
		make build-cmd

build-cmd:
	go build -o bin/tftp -v .
