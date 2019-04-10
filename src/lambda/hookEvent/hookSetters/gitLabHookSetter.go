package hookSetters

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sixleaveakkm/AWSGitHook/src/lambda/hookEvent"
	"github.com/thedevsaddam/gojsonq"
)

type GitLabHookSetter struct {
}

func (setter *GitLabHookSetter) Set(request *events.APIGatewayProxyRequest, hookEventPtr *hookEvent.HookEvent) error {
	hookEventPtr.GitFlavour = "gitlab"
	repoURL := gojsonq.New().JSONString(request.Body).Find("project.http_url")
	if repoURL == nil {
		return errors.New("parse repository failed")
	}
	hookEventPtr.RepositoryName = repoURL.(string)
	//todo: implement
	return nil
}

func (setter *GitLabHookSetter) Match(hookEvent *hookEvent.HookEvent, queueResult *hookEvent.QueueResult) bool {
	//todo: implement
	return false
}
