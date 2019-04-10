package main

import (
	"encoding/json"
	"github.com/sixleaveakkm/AWSGitHook/src/lambda/connect/gitConnector"
	"github.com/sixleaveakkm/AWSGitHook/src/lambda/hookEvent"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	executeType := os.Args[1]

	log.SetFlags(log.Ldate | log.Ltime)
	hookEventFile, err := os.Open("./git_info.json")
	if err != nil {
		log.Fatalf("Open git info file failed, %v", err)
		return
	}
	defer func() {
		if err := hookEventFile.Close(); err != nil {
			log.Fatalf("Error close zip file %v", err)
		}
	}()

	byteValue, _ := ioutil.ReadAll(hookEventFile)
	hookEventData := new(hookEvent.HookEvent)
	err = json.Unmarshal(byteValue, &hookEventData)
	if err != nil {
		log.Fatalf("json Unmarshal data failed, %v", err)
	}

	var gitConn gitConnector.GitConnector
	switch hookEventData.GitFlavour {
	case "bitbucket":
		gitConn = &gitConnector.BitBucketConnector{
			HookEventPtr: hookEventData,
		}
	}
	switch executeType {
	case "build_start":
		gitConn.BuildStart()
	case "build_fail":
		gitConn.BuildFail()
	case "build_succ":
		gitConn.BuildSucc()
	case "cleanComment":
		gitConn.CleanComment()
	case "comment":
		gitConn.Comment(os.Args[2])
	case "cloneURL":
		gitConn.PrintCloneURL()
	case "executePath":
		gitConn.PrintExecutePath()
	}

}
