.PHONY: build clean deploy package

package: build
	cd docker
	$(aws ecr get-login --no-include-email --region ap-northeast-1)
	docker build -t env-tool/auth .
	docker tag env-tool/auth:latest 902887174334.dkr.ecr.ap-northeast-1.amazonaws.com/env-tool/auth:latest
	docker push 902887174334.dkr.ecr.ap-northeast-1.amazonaws.com/env-tool/auth:latest

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/hookReceiver src/lambda/hookReceiver/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/containerExecuter src/lambda/containerExecuter/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/gitConnector src/lambda/connect/main.go
	cp src/script/outer_build.sh bin/outer_build.sh

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
