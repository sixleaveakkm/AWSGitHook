#!/bin/bash


echo "--------------"
cat git_info.json | jq '.credential="****************"'
echo "--------------"

echo "Cloning project"
cloneURL=`./gitConnector cloneURL`
PATH=$PATH:`pwd`
git clone $cloneURL "repo"
git config --global user.name "DevBot"
git config --global user.email "new_auth_dev_bot@worksap.co.jp"

trigger_event=`cat git_info.json | jq .event | sed -ne "s/\"\(.*\)\"/\1/p"`
source_branch=`cat git_info.json | jq .source | sed -ne "s/\"\(.*\)\"/\1/p"`
destination_branch=`cat git_info.json | jq .destination | sed -ne "s/\"\(.*\)\"/\1/p"`

# export varibles to inside bash
HOOK_TRIGGER_EVENT=$trigger_event
echo "SET \"HOOK_TRIGGER_EVENT\": $trigger_event"
HOOK_SOURCE_BRANCH=$source_branch
echo "SET \"HOOK_SOURCE_BRANCH\": $source_branch"
HOOK_DESTINATION_BRANCH=$destination_branch
echo "SET \"HOOK_DESTINATION_BRANCH\": $destination_branch"
HOOK_PR_ID=""

cd repo
echo "Git checkout to $destination_branch"
git checkout $destination_branch

if [[ $trigger_event =~ pullrequest.* ]];then
	HOOK_PR_ID=`cat ../git_info.json | jq .pullRequestContent.pullRequestId | sed -ne "s/\"\(.*\)\"/\1/p"`
	echo "SET \"HOOK_PR_ID\": $HOOK_PR_ID"
	echo "Git merge $source_branch to $destination_branch"
	git fetch origin ${source_branch}:${source_branch}
  git merge $source_branch
fi

cd ..
gitConnector build_start; echo "put build start"
bash_file=`cat ./git_info.json | jq .ExecutePath | sed -ne "s/\"\(.*\)\"/\1/p"`
cd repo
echo "execute \"$bash_file\""
bash -ex $bash_file
code=$?
echo "Execute file exit with code: $code"
cd ..
if (( $code == 0 ));then
    gitConnector build_succ; echo "put build succ"
else
    gitConnector build_fail; echo "put build fail"
fi
exit $code
