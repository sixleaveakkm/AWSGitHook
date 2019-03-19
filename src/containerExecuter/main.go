package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/thedevsaddam/gojsonq"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

type Credential struct {
	Category string `json:"category"` // bitbucket:oauth
	Key      string `json:"key"`
	Secret   string `json:"secret"`
}
type HookEvent struct {
	ProjectName string `json:"projectName"`
	Event             string     `json:"event"`
	SourceBranch      string     `json:"source"`
	DestinationBranch string     `json:"destination"`
	Comment           string     `json:"comment"`
	CommentAuthor     string     `json:"commentAuthor"`
	Uuid              string     `json:"uuid"`
	CommitId          string     `json:"commitId"`
	PullRequestId     string     `json:"pullRequestId"`
	Credential        Credential `json:"credential"`
}

func ContainerExecuter(_ context.Context, request Request) (Response, error) {
	log.Printf("Evnet %+v", request)

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	hookEventPtr, err := json.Unmarshal(_)

	err = ioutil.WriteFile(hookEventPtr.Uuid, jsonStr, 0666)
	if err != nil {
		log.Fatalf("Create file failed: %+v\n", err)
	} else {
		// upload to s3
		uploader := s3manager.NewUploader(sess)
		file, err := os.Open(hookEventPtr.Uuid)
		if err != nil {
			log.Fatal("failed to open uuid file %v", err)
		}

		result, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(os.Getenv("TRIGGERBUCKET")),
			Key:    aws.String(bucketKey),
			Body:   file,
		})
		if err != nil {
			log.Fatal("failed to upload file, %v", err)
		}
		log.Printf("File uploaded to: %s\n", result.Location)

		builder := codebuild.New(sess)
		buildOutput, err != builder.StartBuild(&codebuild.StartBuildInput{
			ArtifactsOverride: &codebuild.ProjectArtifacts{

			},
			ProjectName: hookEventPtr.ProjectName
		})
	}
	}

	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            "",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil

}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	lambda.Start(ContainerExecuter)
}
