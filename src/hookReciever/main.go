package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	. "github.com/sixleaveakkm/AWSGitHook/src"
	"github.com/thedevsaddam/gojsonq"
	"log"
	"strings"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

func getGitHookFlavour(headers map[string]string) (string, bool) {
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

func getRepositoryURL(gitFlavour string, request *Request) (string, bool) {
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

func response_410(reason string) Response {
	return Response{
		StatusCode:      410,
		IsBase64Encoded: false,
		Body:            reason,
		Headers: map[string]string{
			"Content-Type": "application/text",
		},
	}
}

func HookReceiver(ctx context.Context, request Request) (Response, error) {
	log.Printf("Evnet %+v", request)
	gitFlavour, ok := getGitHookFlavour(request.Headers)
	if !ok {
		log.Println("Exit because not a hook")
		return response_410("Not a hook"), nil
	}

	repositoryName, ok := getRepositoryURL(gitFlavour, &request)
	if !ok {
		log.Println("Exit because no repository data found")
		return response_410("Repository data not found"), nil
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
