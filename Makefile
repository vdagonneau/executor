.PHONY: all clean

all: agent executor

clean:
	rm -f agent
	rm -f executor
	rm -f cmd/executor/embed/agent

executor:
	go build ./cmd/executor

agent:
	go build -ldflags="-s -w -linkmode external -extldflags '-static'" ./cmd/agent
	upx -1 agent
	cp "$(shell pwd)/agent" cmd/executor/embed/agent