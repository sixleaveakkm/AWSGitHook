.PHONY: build clean deploy package

package: build
	cd docker
	bash ./build.sh

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/hookReceiver src/lambda/hookReceiver/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/containerExecuter src/lambda/containerExecuter/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/gitConnector src/lambda/connect/main.go
	cp src/script/outer_build.sh bin/outer_build.sh

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
