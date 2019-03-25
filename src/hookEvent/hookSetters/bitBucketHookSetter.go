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

func (setter *BitBucketHookSetter) setPush(request *events.APIGatewayProxyRequest, hookPtr *hookEvent.HookEvent) {
	hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("push.changes[0].new.name").(string)
	log.Println(hookPtr.SourceBranch)
	hookPtr.DestinationBranch = hookPtr.SourceBranch
	log.Println(hookPtr.DestinationBranch)

	hookPtr.CommitURL = gojsonq.New().JSONString(request.Body).Find("push.changes[0].links.commits.href").(string)

}

func (setter *BitBucketHookSetter) setPullRequest(request *events.APIGatewayProxyRequest, hookPtr *hookEvent.HookEvent) {
	hookPtr.SourceBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.branch.name").(string)
	log.Println(hookPtr.SourceBranch)
	hookPtr.DestinationBranch = gojsonq.New().JSONString(request.Body).Find("pullrequest.destination.branch.name").(string)
	log.Println(hookPtr.DestinationBranch)

	hookPtr.CommitURL = gojsonq.New().JSONString(request.Body).Find("pullrequest.source.commit.links.html.href").(string)
	log.Println(hookPtr.CommitURL)
	hookPtr.PullRequestId = fmt.Sprintf("%d", int(gojsonq.New().JSONString(request.Body).Find("pullrequest.id").(float64)))
	log.Println(hookPtr.PullRequestId)

	if gojsonq.New().JSONString(request.Body).Find("comment") != nil {
		hookPtr.Comment = gojsonq.New().JSONString(request.Body).Find("comment.content.raw").(string)
		log.Println(hookPtr.DestinationBranch)
		hookPtr.CommentAuthor = gojsonq.New().JSONString(request.Body).Find("actor.uuid").(string)
		log.Println(hookPtr.CommentAuthor)
	}
}

func (setter *BitBucketHookSetter) Set(request *events.APIGatewayProxyRequest, hookPtr *hookEvent.HookEvent) error {
	hookPtr.GitFlavour = "bitbucket"
	repoURL := gojsonq.New().JSONString(request.Body).Find("repository.links.html.href")
	if repoURL == nil {
		return errors.New("parse repository failed")
	}
	hookPtr.RepositoryName = repoURL.(string)
	log.Printf("RepositoryName is %v", hookPtr.RepositoryName)

	hookPtr.RepositoryShort = gojsonq.New().JSONString(request.Body).Find("repository.full_name").(string)
	log.Println(hookPtr.RepositoryShort)

	eventStr := request.Headers["X-Event-Key"]
	hookPtr.Event = eventStr
	log.Printf("eventStr ,%T, %v", eventStr, eventStr)
	eventArr := strings.Split(eventStr, ":")

	switch eventArr[0] {
	case "repo":
		if eventArr[1] == "push" {
			setter.setPush(request, hookPtr)
		}
	case "pullrequest":
		setter.setPullRequest(request, hookPtr)
	}
	return nil
}

func (setter *BitBucketHookSetter) Match(hookEvent *hookEvent.HookEvent, queueResultList *hookEvent.QueueResult) bool {
	for _, queueResult := range queueResultList.Events {
		hit := false
		switch hookEvent.Event {
		case "repo:push":
			if queueResult.SourceBranch != "" && queueResult.SourceBranch != hookEvent.SourceBranch {
				hit = false
			}
			hit = true
		case "pullrequest:created", "pullrequest:updated", "pullrequest:fulfilled", "pullrequest:rejected":
			if queueResult.DestinationBranch != "" && queueResult.DestinationBranch != hookEvent.DestinationBranch {
				hit = false
			}
			if queueResult.SkipWIP == true && hookEvent.PullRequestTitle[0:4] == "WIP:" {
				hit = false
			}
			hit = true
		case "pullrequest:comment_created":
			if queueResult.DestinationBranch != "" && queueResult.DestinationBranch != hookEvent.DestinationBranch {
				hit = false
			}
			if queueResult.SkipWIP == true && hookEvent.PullRequestTitle[0:4] == "WIP:" {
				hit = false
			}
			if len(queueResult.ReBuildComments) == 0 {
				hit = false
			}
			contains := false
			for _, value := range queueResult.ReBuildComments {
				if value == hookEvent.Comment {
					contains = true
					break
				}
			}
			if contains == false {
				hit = false
			}
			if len(queueResult.PermittedCommentUsers) == 0 {
				hit = true
			}
			contains = false
			for _, value := range queueResult.PermittedCommentUsers {
				if value == hookEvent.CommentAuthor {
					hit = true
				}
			}
			hit = false

		}
		if hit {
			return true
		}
	}
	return false
}
