package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	lambdaSDK "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent/hookSetters"
	"github.com/thedevsaddam/gojsonq"
	"log"
	"os"
	"strings"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

func isHookRegistered(repositoryName string, hookPtr *hookEvent.HookEvent, sess *session.Session) bool {
	svc := dynamodb.New(sess)
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Repository": {
				S: aws.String(repositoryName),
			},
			"Events": {
				S: aws.String(hookPtr.Event),
			},
		},
		TableName: aws.String(os.Getenv("TABLENAME")),
	}
	log.Printf("Queue info: \n\tRepository: %s\n\tEvents: %s\n\tTableName: %s", repositoryName, hookPtr.Event, os.Getenv("TABLENAME"))
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
		return false
	}
	log.Printf("result: %v", result)
	items := result.Item

	if items == nil {
		log.Println("No matching repo data")
		return false
	}

	// matching pattern
	switch hookPtr.Event {
	case "repo:push":
		if items["SourceBranch"] != nil && *items["SourceBranch"].S == hookPtr.SourceBranch {
			hookPtr.Credential.Category = *items["CredCategory"].S
			hookPtr.Credential.Key = *items["CredKey"].S
			hookPtr.Credential.Secret = *items["CredSecret"].S
			hookPtr.ProjectName = *items["ProjectName"].S
			//hookPtr.ExecutePath = *items["ExecutePath"].S
			return true
		}
		return false
	case "pullrequest:created", "pullrequset:updated", "pullrequest:fulfilled", "pullrequest:rejected":
		if items["DestinationBranch"] != nil && *items["DestinationBranch"].S != hookPtr.DestinationBranch {
			return false
		}

		hookPtr.Credential.Category = *items["CredCategory"].S
		hookPtr.Credential.Key = *items["CredKey"].S
		hookPtr.Credential.Secret = *items["CredSecret"].S
		hookPtr.ProjectName = *items["ProjectName"].S
		//hookPtr.ExecutePath = *items["ExecutePath"].S
		return true

	case "pullrequest:comment_created", "pullrequest:comment_updated", "pullrequest:comment_deleted":
		if items["DestinationBranch"] != nil && *items["DestinationBranch"].S != hookPtr.DestinationBranch {
			return false
		}
		if items["Comment"] != nil {
			isContaining := false
			for _, valuePtr := range items["Comment"].L {
				if *valuePtr.S == hookPtr.Comment {
					isContaining = true
					break
				}
			}
			if !isContaining {
				return false
			}
		}
		if items["CommentAuthor"] != nil {
			isContaining := false
			for _, valuePtr := range items["CommentAuthor"].L {
				if *valuePtr.S == hookPtr.Comment {
					isContaining = true
					break
				}
			}
			if !isContaining {
				return false
			}
		}
		hookPtr.Credential.Category = *items["CredCategory"].S
		hookPtr.Credential.Key = *items["CredKey"].S
		hookPtr.Credential.Secret = *items["CredSecret"].S
		hookPtr.ProjectName = *items["ProjectName"].S
		//hookPtr.ExecutePath = *items["ExecutePath"].S
		return true
	default:
		log.Println("hook event no match")
		return false
	}
}

func getHookEvent(gitFlavour string, request *Request) (*hookEvent.HookEvent, bool) { // if ok
	hookPtr := new(hookEvent.HookEvent)
	log.Printf("request.Body T: %T\n", request.Body)
	switch gitFlavour {
	case "bitbucket":

	}
	if hookPtr != nil {
		return hookPtr, true
	} else {
		return nil, false
	}
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
	//log.Printf("Evnet %+v", request)
	gitFlavour, ok := getGitHookFlavour(request.Headers)
	if !ok {
		log.Println("Exit because not a hook")
		return response410("Not a hook"), nil
	}
	log.Printf("Git flavour is %v\n", gitFlavour)

	hookEventPtr := hookEvent.HookEvent{}
	var hookSetter hookEvent.HookSetter
	switch gitFlavour {
	case "bitbucket":
		hookSetter = new(hookSetters.GitHubHookSetter)
	}
	err := hookSetter.Set(&request, &hookEventPtr)

	hookEventPtr, ok := getHookEvent(gitFlavour, &request)
	if !ok {
		log.Println("Exit because no hook data found")
		return response410("Hook data not URL found"), nil
	}
	log.Printf("hookEvent: %+v\n", hookEventPtr)
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	if !isHookRegistered(repositoryName, hookEventPtr, sess) {
		return Response{
			StatusCode:      204,
			IsBase64Encoded: false,
			Body:            "Pattern not match",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}
	hookEventPtr.GitFlavour = gitFlavour

	// export zipped json to file: UUid
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
