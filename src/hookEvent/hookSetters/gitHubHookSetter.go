package hookSetters

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"github.com/thedevsaddam/gojsonq"
)

type GitHubHookSetter struct {
}

func (setter *GitHubHookSetter) Set(request *events.APIGatewayProxyRequest, hookEventPtr *hookEvent.HookEvent) error {
	hookEventPtr.GitFlavour = "bitbucket"
	repoURL := gojsonq.New().JSONString(request.Body).Find("repository.achieve_url")
	if repoURL == nil {
		return errors.New("parse repository failed")
	}
	hookEventPtr.RepositoryName = repoURL.(string)
	//todo: implement
	return nil
}

func (setter *GitHubHookSetter) Match(hookEvent *hookEvent.HookEvent, result *hookEvent.QueueResult) bool {
	//todo: implement
	return false
}
