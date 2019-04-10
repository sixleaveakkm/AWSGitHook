#!/bin/bash
cp ../src/script/outer_build.sh .
cp ../bin/gitConnector .
$(aws ecr get-login --no-include-email --region ap-northeast-1)
docker build -t env-tool/auth .
docker tag env-tool/auth:latest 902887174334.dkr.ecr.ap-northeast-1.amazonaws.com/env-tool/auth:latest
docker push 902887174334.dkr.ecr.ap-northeast-1.amazonaws.com/env-tool/auth:latest
