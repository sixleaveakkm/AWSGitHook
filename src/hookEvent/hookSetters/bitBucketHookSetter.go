package hookSetters

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"github.com/thedevsaddam/gojsonq"
	"log"
	"strings"
)

type BitBucketHookSetter struct {
}

func (setter *BitBucketHookSetter) Set(request *events.APIGatewayProxyRequest, hookPtr *hookEvent.HookEvent) error {
	hookPtr.GitFlavour = "bitbucket"
	repoURL := gojsonq.New().JSONString(request.Body).Find("repository.links.html.href")
	if repoURL == nil {
		return errors.New("parse repository failed")
	}
	hookPtr.RepositoryName = repoURL.(string)
	log.Println("repositoryName is %v", hookPtr.RepositoryName)

	log.Printf("request.Headers[x-event-key] T: %T\n", request.Headers["X-Event-Key"])
	eventStr := request.Headers["X-Event-Key"]
	hookPtr.Event = eventStr
	log.Printf("eventStr ,%T, %v", eventStr, eventStr)
	eventArr := strings.Split(eventStr, ":")

	switch eventArr[0] {
	case "repo":
		if eventArr[1] == "push" {
			hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("push.changes[0].new.name").(string)
			log.Println(hookPtr.SourceBranch)
			hookPtr.DestinationBranch = hookPtr.SourceBranch
			log.Println(hookPtr.DestinationBranch)
			log.Printf("x-request-uuid T: %T", request.Headers["X-Request-UUID"])
			hookPtr.Uuid = request.Headers["X-Request-UUID"]
			hookPtr.CommitURL = gojsonq.New().JSONString(request.Body).Find("push.changes[0].links.commits.href").(string)
			hookPtr.RepositoryShort = gojsonq.New().JSONString(request.Body).Find("repository.full_name").(string)
		}
	case "pullrequest":
		hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.branch.name").(string)
		log.Println(hookPtr.SourceBranch)
		hookPtr.DestinationBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.destination.branch.name").(string)
		log.Println(hookPtr.DestinationBranch)
		hookPtr.Uuid = request.Headers["X-Request-UUID"]
		log.Println(hookPtr.Uuid)
		hookPtr.CommitURL = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.commit.links.html.href").(string)
		log.Println(hookPtr.CommitURL)
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
