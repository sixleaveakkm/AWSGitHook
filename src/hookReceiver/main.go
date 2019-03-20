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
	"github.com/thedevsaddam/gojsonq"
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
	RepositoryName    string     `json:"repositoryName"`
	RepositoryShort   string     `json:"reposirotyShort"`
	GitFlavour        string     `json:"gitFlavour"`
	ProjectName       string     `json:"projectName"`
	Event             string     `json:"event"`
	SourceBranch      string     `json:"source"`
	DestinationBranch string     `json:"destination"`
	Comment           string     `json:"comment"`
	CommentAuthor     string     `json:"commentAuthor"`
	Uuid              string     `json:"uuid"`
	CommitId          string     `json:"commitId"`
	PullRequestId     string     `json:"pullRequestId"`
	Credential        Credential `json:"credential"`
	ExecutePath       string     `json:"ExecutePath"`
}

func isHookRegistered(repositoryName string, hookPtr *HookEvent, sess *session.Session) bool {
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

func getHookEvent(gitFlavour string, request *Request) (*HookEvent, bool) { // if ok
	hookPtr := new(HookEvent)
	log.Printf("request.Body T: %T\n", request.Body)
	switch gitFlavour {
	case "bitbucket":
		log.Printf("request.Headers[x-event-key] T: %T\n", request.Headers["X-Event-Key"])
		eventStr := request.Headers["X-Event-Key"]
		log.Printf("eventStr ,%T, %v", eventStr, eventStr)
		eventArr := strings.Split(eventStr, ":")
		switch eventArr[0] {
		case "repo":
			if eventArr[1] == "push" {
				hookPtr.Event = eventStr
				log.Println(hookPtr.Event)
				hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("push.changes[0].new.name").(string)
				log.Println(hookPtr.SourceBranch)
				hookPtr.DestinationBranch = hookPtr.SourceBranch
				log.Println(hookPtr.DestinationBranch)
				log.Printf("x-request-uuid T: %T", request.Headers["X-Request-UUID"])
				hookPtr.Uuid = request.Headers["X-Request-UUID"]
				hookPtr.CommitId = gojsonq.New().JSONString(request.Body).Find("push.changes[0].links.commits.href").(string)
				hookPtr.RepositoryShort = gojsonq.New().JSONString(request.Body).Find("repository.full_name").(string)
			}
		case "pullrequest":
			hookPtr.Event = eventStr
			log.Println(hookPtr.Event)
			hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.branch.name").(string)
			log.Println(hookPtr.SourceBranch)
			hookPtr.DestinationBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.destination.branch.name").(string)
			log.Println(hookPtr.DestinationBranch)
			hookPtr.Uuid = request.Headers["X-Request-UUID"]
			log.Println(hookPtr.Uuid)
			hookPtr.CommitId = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.commit.links.html.href").(string)
			log.Println(hookPtr.CommitId)
			hookPtr.PullRequestId = fmt.Sprintf("%d", int(gojsonq.New().JSONString(request.Body).Find("pullrequest.id").(float64)))
			log.Println(hookPtr.PullRequestId)
			hookPtr.RepositoryShort = gojsonq.New().JSONString(request.Body).Find("repository.full_name").(string)
			log.Println(hookPtr.RepositoryShort)
			if gojsonq.New().JSONString(request.Body).Find("comment") != nil {
				hookPtr.Comment = gojsonq.New().JSONString(request.Body).Find("comment.content.raw").(string)
				log.Println(hookPtr.DestinationBranch)
				hookPtr.CommentAuthor = gojsonq.New().JSONString(request.Body).Find("actor.uuid").(string)
				log.Println(hookPtr.CommentAuthor)
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
	switch gitFlavour {
	case "githubent", "github":
		return gojsonq.New().JSONString(request.Body).Find("repository.archieve_url").(string), true
	case "gitlab":
		return gojsonq.New().JSONString(request.Body).Find("project.http_url").(string), true
	case "bitbucket":
		return gojsonq.New().JSONString(request.Body).Find("repository.links.html.href").(string), true
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
	//log.Printf("Evnet %+v", request)
	gitFlavour, ok := getGitHookFlavour(request.Headers)
	if !ok {
		log.Println("Exit because not a hook")
		return response410("Not a hook"), nil
	}
	log.Printf("Git flavour is %v\n", gitFlavour)

	repositoryName, ok := getRepositoryURL(gitFlavour, &request)
	if !ok {
		log.Println("Exit because no repository data found")
		return response410("Repository data not found"), nil
	}
	log.Printf("Repo name is %v\n", repositoryName)

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
	hookEventPtr.RepositoryName = repositoryName
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
