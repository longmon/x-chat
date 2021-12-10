APP=x-chat

.PHONY: build
build:
	CGO_ENABLED=0;go build -o ${APP} *.go

.PHONY: dev
dev:
	go run *.go

