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
		hookPtr.CommentAuthor = gojsonq.New().JSONString(request.Body).Find("actor.username").(string)
		log.Println(hookPtr.CommentAuthor)
	}
}

func (setter *BitBucketHookSetter) Set(request *events.APIGatewayProxyRequest, hookPtr *hookEvent.HookEvent) error {
	hookPtr.GitFlavour = "bitbucket"

	repoURL := gojsonq.New().JSONString(request.Body).Find("repository.links.html.href")
	if repoURL == nil {
		return errors.New("parse repository failed, request is : " + fmt.Sprintf("%v", request))
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

func (setter *BitBucketHookSetter) matchEach(hookEvent *hookEvent.HookEvent, queueResult hookEvent.Event) bool {
	switch hookEvent.Event {
	case "repo:push":
		if queueResult.SourceBranch != "" && queueResult.SourceBranch != hookEvent.SourceBranch {
			log.Printf("push source branch not match")
			return false
		}
		return true
	case "pullrequest:created", "pullrequest:updated", "pullrequest:fulfilled", "pullrequest:rejected":
		if queueResult.DestinationBranch != "" && queueResult.DestinationBranch != hookEvent.DestinationBranch {
			log.Printf("pullrequest destination branch not match")
			return false
		}
		if queueResult.SkipWIP == true && hookEvent.PullRequestTitle[0:4] == "WIP:" {
			log.Printf("SkipWIP is true, and title starts with 'WIP:'")
			return false
		}
		return true
	case "pullrequest:comment_created":
		if queueResult.DestinationBranch != "" && queueResult.DestinationBranch != hookEvent.DestinationBranch {
			log.Printf("comment created, destination branch not match")
			return false
		}
		if queueResult.SkipWIP == true && hookEvent.PullRequestTitle[0:4] == "WIP:" {
			log.Printf("SkipWIP is true, and title starts with 'WIP:'")
			return false
		}
		if len(queueResult.ReBuildComments) == 0 {
			log.Printf("no rebuild comment set")
			return false
		}
		contains := false
		for _, value := range queueResult.ReBuildComments {
			if value == hookEvent.Comment {
				contains = true
				log.Printf("comment found a match")
				break
			}
		}
		if contains == false {
			log.Printf("Rebuild comment no match")
			return false
		}
		if len(queueResult.PermittedCommentUsers) == 0 {
			log.Printf("comment has a match and permitted user not set")
			return true
		}
		contains = false
		for _, value := range queueResult.PermittedCommentUsers {
			if value == hookEvent.CommentAuthor {
				log.Printf("comment match and user found match")
				return true
			}
		}
		log.Printf("comment match but user not match")
		return false
	default:
		return false
	}

}

func (setter *BitBucketHookSetter) Match(hookEvent *hookEvent.HookEvent, queueResultList *hookEvent.QueueResult) bool {
	for _, queueResult := range queueResultList.Events {
		hit := setter.matchEach(hookEvent, queueResult)
		if hit {
			return true
		}
	}
	return false
}
