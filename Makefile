.PHONY: build clean deploy package

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/hookReceiver src/hookReceiver/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/containerExecuter src/containerExecuter/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/gitConnector src/connect/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
