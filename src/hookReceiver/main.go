package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	lambdaSDK "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent/hookSetters"
	"log"
	"os"
	"strings"
)

type Response events.APIGatewayProxyResponse

func queueHookRegisteredInfo(repositoryName string, event string, sess *session.Session) (*hookEvent.QueueResult, error) {
	svc := dynamodb.New(sess)
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Repository": {
				S: aws.String(repositoryName),
			},
			"Events": {
				S: aws.String(event),
			},
		},
		TableName: aws.String(os.Getenv("TABLENAME")),
	}
	result, err := svc.GetItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				log.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				log.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, errors.New("queue database failed")
	}
	items := result.Item

	if items == nil {
		return nil, errors.New("no match data")
	}

	queueResult := new(hookEvent.QueueResult)
	if items["ProjectName"] != nil {
		queueResult.ProjectName = *items["ProjectName"].S
		log.Printf("ProjectName set, queueResult: %+v", queueResult)
	}
	queueResult.Credential = hookEvent.Credential{}
	queueResult.Events = []hookEvent.Event{}
	if items["CredCategory"] != nil {
		queueResult.Credential.Category = *items["CredCategory"].S
		log.Printf("CredCategory set, queueResult: %+v", queueResult)
	}
	if items["CredKey"] != nil {
		queueResult.Credential.Key = *items["CredKey"].S
		log.Printf("Key set, queueResult: %+v", queueResult)
	}
	if items["CredSecret"] != nil {
		queueResult.Credential.Secret = *items["CredSecret"].S
		log.Printf("Secret set, queueResult: %+v", queueResult)
	}
	if items["ExecutePath"] != nil {
		queueResult.ExecutePath = *items["ExecutePath"].S
	}

	log.Printf("Before loop")
	for _, event := range items["Triggers"].L {
		value := event.M
		eventStruct := new(hookEvent.Event)
		if value["SkipWIP"] != nil {
			eventStruct.SkipWIP = *value["SkipWIP"].S == "true"
		}
		if value["PermittedCommentUsers"] != nil {
			var users []string
			for _, userList := range value["PermittedCommentUsers"].L {
				users = append(users, *userList.S)
			}
			eventStruct.PermittedCommentUsers = users
		}
		if value["Comments"] != nil {
			var comments []string
			for _, commentList := range value["Comments"].L {
				comments = append(comments, *commentList.S)
			}
			eventStruct.ReBuildComments = comments
		}
		if value["SourceBranch"] != nil {
			eventStruct.SourceBranch = *value["SourceBranch"].S
		}
		if value["DestinationBranch"] != nil {
			eventStruct.DestinationBranch = *value["DestinationBranch"].S
		}

		queueResult.Events = append(queueResult.Events, *eventStruct)
	}
	log.Printf("queueResult: %+v", queueResult)
	return queueResult, nil
}

func getGitHookFlavour(headers map[string]string) (string, error) {
	if _, ok := headers["X-Hub-Signature"]; ok {
		return "githubent", nil
	}
	if _, ok := headers["X-Gitlab-Event"]; ok {
		return "gitlab", nil
	}
	if agent, ok := headers["User-Agent"]; ok {
		if strings.HasPrefix(agent, "Bitbucket-Webhooks") {
			return "bitbucket", nil
		} else if strings.HasPrefix(agent, "GitHub-Hookshot") {
			return "github", nil
		}
	}
	return "", errors.New("can't identify git flavour")
}

func response410(reason string) Response {
	return Response{
		StatusCode:      410,
		IsBase64Encoded: false,
		Body:            reason,
		Headers: map[string]string{
			"Content-Type": "application/text",
		},
	}
}

func HookReceiver(_ context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	log.Printf("Event: %v", request.Headers["X-Request_UUID"])

	gitFlavour, err := getGitHookFlavour(request.Headers)
	if err != nil {
		log.Printf("Error, %v", err)
		return response410(""), nil
	}
	log.Printf("Git flavour is %v\n", gitFlavour)

	hookEventPtr := new(hookEvent.HookEvent)
	var hookSetter hookEvent.HookSetter
	switch gitFlavour {
	case "bitbucket":
		hookSetter = new(hookSetters.BitBucketHookSetter)
	default:
		log.Fatalf("Unknown git flavour")
	}
	err = hookSetter.Set(&request, hookEventPtr)
	if err != nil {
		log.Printf("Set hook failed, %v", err)
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	registeredInfo, err := queueHookRegisteredInfo(hookEventPtr.RepositoryName, hookEventPtr.Event, sess)
	if err != nil {
		return response410("repository not registered"), nil
	}
	if !hookSetter.Match(hookEventPtr, registeredInfo) {
		return Response{
			StatusCode:      204,
			IsBase64Encoded: false,
			Body:            "Pattern not match",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}
	hookEventPtr.ProjectName = registeredInfo.ProjectName
	hookEventPtr.Credential = registeredInfo.Credential
	hookEventPtr.ExecutePath = registeredInfo.ExecutePath
	jsonStr, err := json.Marshal(hookEventPtr)

	if err != nil {
		log.Fatalf("Error when marshal json: %+v\n", err)
	} else {
		log.Printf("jsonStr is : %s", jsonStr)
		lambdaExecutor := lambdaSDK.New(sess)
		invocationType := "Event"
		functionName := os.Getenv("CONTAINER_EXECUTER_NAME")
		_, err := lambdaExecutor.Invoke(&lambdaSDK.InvokeInput{
			FunctionName:   &functionName,
			InvocationType: &invocationType,
			Payload:        jsonStr,
		})
		if err != nil {
			log.Fatalf("Error invoke function: %v", err)
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
	lambda.Start(HookReceiver)
}
