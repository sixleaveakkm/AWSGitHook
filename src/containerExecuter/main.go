package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func createFile(hookEventPtr *hookEvent.HookEvent) {
	bytes, err := json.Marshal(hookEventPtr)
	if err != nil {
		log.Fatalf("Error marshal data %v", err)
	}
	err = ioutil.WriteFile("/tmp/git_info.json", bytes, 0666)
	if err != nil {
		log.Fatalf("Error create file %v", err)
	}
	zipFile, err := os.Create("/tmp/git_info.json.zip")
	if err != nil {
		log.Fatalf("Error create zip file %v", err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			log.Fatalf("Error close zip file %v", err)
		}
	}()
	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Fatalf("Error close zip writer %v", err)
		}
	}()
	file, err := os.Open("/tmp/git_info.json")
	if err != nil {
		log.Fatalf("Error read json file %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Error close json file %v", err)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		log.Fatalf("Error read file stat, %v", err)
	}
	log.Printf("config.json length is %d bytes", info.Size())
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		log.Fatalf("Error read file header %v", err)
	}
	header.Name = "config.json"
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		log.Fatalf("Error create header file %v", err)
	}

	_, err = io.Copy(writer, file)
	if err != nil {
		log.Fatalf("Error copy file %v", err)
	}
}
func ContainerExecuter(ctx context.Context, hookEventPtr hookEvent.HookEvent) error {
	log.Printf("payload: %v", hookEventPtr)

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
		hookEventPtr.RepositoryName[8:],
		eventKey,
		"git_info.json.zip",
	}, "/")

	log.Printf("bucket key: %s", bucketKey)

	createFile(&hookEventPtr)
	// upload to s3
	uploader := s3manager.NewUploader(sess)
	uploadFile, err := os.Open("/tmp/git_info.json.zip")
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
