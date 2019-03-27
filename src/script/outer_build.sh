#!/bin/bash


echo "--------------"
cat git_info.json | jq '.credential="****************"'
echo "--------------"

cloneURL=`./gitConnector cloneURL`
PATH=$PATH:`pwd`
git clone $cloneURL "repo"
git config --global user.name "DevBot"
git config --global user.email "new_auth_dev_bot@worksap.co.jp"

trigger_event=`cat git_info.json | jq .event | sed -ne "s/\"\(.*\)\"/\1/p"`
source_branch=`cat git_info.json | jq .source | sed -ne "s/\"\(.*\)\"/\1/p"`
destination_branch=`cat git_info.json | jq .destination | sed -ne "s/\"\(.*\)\"/\1/p"`
cd repo
git checkout $destination_branch
if [[ $trigger_event =~ pullrequest.* ]];then
	git fetch origin ${source_branch}:${source_branch}
    git merge $source_branch
fi

cd ..
gitConnector build_start; echo "put build start"
bash_file=`cat ./git_info.json | jq .ExecutePath | sed -ne "s/\"\(.*\)\"/\1/p"`
cd repo
bash -x $bash_file
code=$?
cd ..
if (( $code == 0 ));then
    gitConnector build_succ; echo "put build succ"
else
    gitConnector build_fail; echo "put build fail"
fi
exit $code