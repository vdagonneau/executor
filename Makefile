.PHONY: all clean

all: agent executor

clean:
	rm -f agent
	rm -f executor
	rm -f cmd/executor/embed/agent

executor:
	go build -ldflags "-X main.commitHash=$(shell git rev-parse HEAD)" ./cmd/executor

agent:
	go build -ldflags="-X main.commitHash=$(shell git rev-parse HEAD) -s -w -linkmode external -extldflags '-static'" ./cmd/agent
	upx -1 agent
	mkdir -p cmd/executor/embed
	cp "$(shell pwd)/agent" cmd/executor/embed/agent