#!/bin/bash

echo "git clone"
cloneURL=`./gitConnector cloneURL`
git clone $cloneURL "repo"
git config --global user.name "DevBot"
git config --global user.email "new_auth_dev_bot@worksap.co.jp"

trigger_event=`cat git_info.json | jq .event`
source_branch=`cat git_info.json | jq .source`
destination_branch=`cat git_info.json | jq .destination`
cd repo
git checkout $destination_branch
if [[ $trigger_event =~ pullrequest.* ]];then
    git merge $source_branch
fi

PATH=$PATH:`pwd`

./gitConnector build_start


bash -x `cat ../git_info.json | jq .ExecutePath`
code=$?
if (( $? == 0 ));then
    gitConnector build_succ
else
    gitConnector build_fail
fi
exit $code