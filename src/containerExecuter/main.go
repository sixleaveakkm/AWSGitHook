package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sixleaveakkm/AWSGitHook/src/connect/gitConnector"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

func createFile(hookEventPtr *hookEvent.HookEvent) {
	bytes, err := json.Marshal(hookEventPtr)
	if err != nil {
		log.Fatalf("Error marshal data %v", err)
	}
	err = ioutil.WriteFile("git_info.json", bytes, 0666)
	if err != nil {
		log.Fatalf("Error create file %v", err)
	}
	//zipFile, err := os.Create("/tmp/package.zip")
	//if err != nil {
	//	log.Fatalf("Error create zip file %v", err)
	//}
	//defer func() {
	//	if err := zipFile.Close(); err != nil {
	//		log.Fatalf("Error close zip file %v", err)
	//	}
	//}()
	//zipWriter := zip.NewWriter(zipFile)
	//defer func() {
	//	if err := zipWriter.Close(); err != nil {
	//		log.Fatalf("Error close zip writer %v", err)
	//	}
	//}()
	//file, err := os.Open("/tmp/git_info.json")
	//if err != nil {
	//	log.Fatalf("Error read json file %v", err)
	//}
	//defer func() {
	//	if err := file.Close(); err != nil {
	//		log.Fatalf("Error close json file %v", err)
	//	}
	//}()
	//info, err := file.Stat()
	//if err != nil {
	//	log.Fatalf("Error read file stat, %v", err)
	//}
	//log.Printf("config.json length is %d bytes", info.Size())
	//header, err := zip.FileInfoHeader(info)
	//if err != nil {
	//	log.Fatalf("Error read file header %v", err)
	//}
	//header.Name = "config.json"
	//header.Method = zip.Deflate
	//
	//writer, err := zipWriter.CreateHeader(header)
	//if err != nil {
	//	log.Fatalf("Error create header file %v", err)
	//}
	//
	//_, err = io.Copy(writer, file)
	//if err != nil {
	//	log.Fatalf("Error copy file %v", err)
	//}
	_, err = exec.Command("zip", "../package.zip", "*").Output()
	if err != nil {
		log.Printf("Zip file failed, %v", err)
	}
}
func ContainerExecuter(ctx context.Context, hookEventPtr hookEvent.HookEvent) error {
	path := hookEventPtr.RepositoryName[8:]

	//clone git
	var gitConn gitConnector.GitConnector
	switch hookEventPtr.GitFlavour {
	case "bitbucket":
		gitConn = &gitConnector.BitBucketConnector{
			HookEventPtr: &hookEventPtr,
		}
		gitConn.Initialize()
		token := gitConn.GetToken()
		_, err := exec.Command("git", "clone", "https://x-token-auth:"+token+"@"+path+".git", "/tmp/"+path).Output()
		if err != nil {
			log.Fatalf("Clone git to tmp failed, %v", err)
		}

	}
	//modify git dir
	_, err := exec.Command("cd", "/tmp/"+path).Output()
	if err != nil {
		log.Fatalf("Change directory failed, %v", err)
	}
	_, err = exec.Command("git", "checkout", hookEventPtr.DestinationBranch).Output()
	if err != nil {
		log.Fatalf("Git checkout to destination branch '%s' failed, %v", hookEventPtr.DestinationBranch, err)
	}
	_, err = exec.Command("git", "merge", hookEventPtr.SourceBranch).Output()
	if err != nil {
		log.Fatalf("Git merge '%s' to '%s' failed, %v", hookEventPtr.SourceBranch, hookEventPtr.DestinationBranch, err)
	}

	createFile(&hookEventPtr)

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	eventKey := ""
	eventArr := strings.Split(hookEventPtr.Event, ":")
	if eventArr[0] == "pullrequest" {
		eventKey = "pullrequest-" + hookEventPtr.PullRequestId
	} else if eventArr[1] == "push" {
		eventKey = "push-" + hookEventPtr.SourceBranch
	}
	bucketKey := strings.Join([]string{
		path,
		eventKey,
		"package.zip",
	}, "/")

	log.Printf("bucket key: %s", bucketKey)

	// upload to s3
	uploader := s3manager.NewUploader(sess)
	uploadFile, err := os.Open("/tmp/package.zip")
	if err != nil {
		log.Fatalf("failed to open uuid file %v", err)
	}
	defer func() {
		if err := uploadFile.Close(); err != nil {
			log.Fatalf("upload file close failed, %v", uploadFile)
		}
	}()

	fi, err := uploadFile.Stat()
	if err != nil {
		log.Fatalf("get stat failed, %v", err)
	}
	log.Printf("file size is : %d bytes", fi.Size())

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("TRIGGERBUCKET")),
		Key:    aws.String(bucketKey),
		Body:   uploadFile,
	})
	if err != nil {
		log.Fatalf("failed to upload file, %v", err)
	}
	log.Printf("File uploaded to: %s\n", result.Location)

	key := os.Getenv("TRIGGERBUCKET") + "/" + bucketKey
	builder := codebuild.New(sess)
	_, err = builder.StartBuild(&codebuild.StartBuildInput{
		SourceLocationOverride: &key,
		ProjectName:            &hookEventPtr.ProjectName,
	})
	if err != nil {
		log.Fatalf("err when start build , %v", err)
	}

	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	lambda.Start(ContainerExecuter)
}
