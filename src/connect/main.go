package main

import (
	"encoding/json"
	gitConnectorImpl "github.com/sixleaveakkm/AWSGitHook/src/connect/gitConnector"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"io/ioutil"
	"log"
	"os"
)

type GitConnector interface {
	Connect()
	BuildStart()
	BuildFail()
	BuildSucc()
	BuildStop()
	Comment(string)
}

func main() {
	executeType := os.Args[1]

	log.SetFlags(log.Ldate | log.Ltime)
	codebuildSrcDir := os.Getenv("CODEBUILD_SRC_DIR")
	hookEventFile, err := os.Open(codebuildSrcDir + "/git_info.json")
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

	var gitConnector GitConnector
	switch hookEventData.GitFlavour {
	case "bitbucket":
		gitConnector = new(gitConnectorImpl.BitBucketConnector{
			HookEventPtr: hookEventData,
		})
	}
	switch executeType {
	case "build_start":
		gitConnector.BuildStart()
	case "build_fail":
		gitConnector.BuildFail()
	case "build_succ":
		gitConnector.BuildSucc()
	case "comment":
		gitConnector.Comment(os.Args[2])
	default:
		gitConnector.Connect()
	}
}
