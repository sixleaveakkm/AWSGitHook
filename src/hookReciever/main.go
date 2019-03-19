package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/thedevsaddam/gojsonq"
	"log"
	"os"
	"strings"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

type HookEvent struct {
	Event             string
	SourceBranch      string
	DestinationBranch string
	Comment           string
	CommentAuthor     string
}

func isHookRegistered(repositoryName string, hookPtr *HookEvent) bool {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Repository": {
				S: aws.String(repositoryName),
			},
			"Event": {
				S: aws.String(hookPtr.Event),
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
		return false
	}
	items := result.Item
	if items == nil {
		log.Println("No matching repo data")
		return false
	}

	// matching pattern
	switch hookPtr.Event {
	case "repo:push":
		if items["SourceBranch"] != nil && *items["SourceBranch"].S == hookPtr.SourceBranch {
			return true
		}
		return false
	case "pullrequest:created", "pullrequset:updated", "pullrequest:fulfilled", "pullrequest:rejected":
		if items["DestinationBranch"] == nil {
			// no branch specify
			return true
		}
		if *items["DestinationBranch"].S == hookPtr.DestinationBranch {
			return true
		} else {
			return false
		}
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
		return true
	default:
		log.Println("hook event no match")
		return false
	}
}

func getHookEvent(gitFlavour string, request *Request) (*HookEvent, bool) { // if ok
	hookPtr := new(HookEvent)
	jsonQ := gojsonq.New().JSONString(request.Body)
	switch gitFlavour {
	case "bitbucket":
		eventStr := request.Headers["X-Event-Key"]
		eventArr := strings.Split(eventStr, ":")
		switch eventArr[0] {
		case "repo":
			if eventArr[1] == "push" {
				hookPtr.Event = eventStr
				hookPtr.SourceBranch = jsonQ.Find("new.name").(string)
				hookPtr.DestinationBranch = hookPtr.SourceBranch
			}
		case "pullrequest":
			hookPtr.Event = eventStr
			hookPtr.SourceBranch = jsonQ.Find("repository.source.branch").(string)
			hookPtr.DestinationBranch = jsonQ.Find("repository.destination.branch").(string)
			if jsonQ.Find("comment") != nil {
				hookPtr.Comment = jsonQ.Find("comment.content.raw").(string)
				hookPtr.CommentAuthor = jsonQ.Find("actor.uuid").(string)
			}
		}

	}
	if hookPtr != nil {
		return hookPtr, true
	} else {
		return nil, false
	}
}

func getGitHookFlavour(headers map[string]string) (string, bool) { //if ok
	if _, ok := headers["X-Hub-Signature"]; ok {
		return "githubent", true
	}
	if _, ok := headers["X-Gitlab-Event"]; ok {
		return "gitlab", true
	}
	if agent, ok := headers["User-Agent"]; ok {
		if strings.HasPrefix(agent, "Bitbucket-Webhooks") {
			return "bitbucket", true
		} else if strings.HasPrefix(agent, "GitHub-Hookshot") {
			return "github", true
		}
	}
	return "", false
}

func getRepositoryURL(gitFlavour string, request *Request) (string, bool) { // if ok
	bodyJsonQ := gojsonq.New().JSONString(request.Body)
	switch gitFlavour {
	case "githubent", "github":
		return bodyJsonQ.Find("repository.archieve_url").(string), true
	case "gitlab":
		return bodyJsonQ.Find("project.http_url").(string), true
	case "bitbucket":
		return bodyJsonQ.Find("repository.links.html").(string), true
	default:
		return "", false
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

func HookReceiver(_ context.Context, request Request) (Response, error) {
	log.Printf("Evnet %+v", request)
	gitFlavour, ok := getGitHookFlavour(request.Headers)
	if !ok {
		log.Println("Exit because not a hook")
		return response410("Not a hook"), nil
	}

	repositoryName, ok := getRepositoryURL(gitFlavour, &request)
	if !ok {
		log.Println("Exit because no repository data found")
		return response410("Repository data not found"), nil
	}

	hookEventPtr, ok := getHookEvent(gitFlavour, &request)
	if !ok {
		log.Println("Exit because no hook data found")
		return response410("Hook data not found"), nil
	}

	if !isHookRegistered(repositoryName, hookEventPtr) {
		return Response{
			StatusCode:      204,
			IsBase64Encoded: false,
			Body:            "",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
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
