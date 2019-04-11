package hookEvent

import (
	"github.com/aws/aws-lambda-go/events"
)

type Credential struct {
	Category string `json:"category"` // bitbucket:oauth
	Key      string `json:"key"`
	Secret   string `json:"secret"`
}

type Event struct {
	SourceBranch          string   `json:"sourceBranch"`
	DestinationBranch     string   `json:"destinationBranch"`
	SkipWIP               bool     `json:"skipWIP"`
	ReBuildComments       []string `json:"rebuildComments"`
	PermittedCommentUsers []string `json:"permittedCommentUsers"`
}
type QueueResult struct {
	Events     []Event
	Credential Credential
	Handler    HookHandler
}
type PullRequestContent struct {
	Comment          string `json:"comment"`
	CommentAuthor    string `json:"commentAuthor"`
	PullRequestId    string `json:"pullRequestId"`
	PullRequestTitle string `json:"pullRequestTitle"`
}

type HookEvent struct {
	GitFlavour        string `json:"gitFlavour"`
	RepositoryName    string `json:"repositoryName"`
	RepositoryShort   string `json:"repositoryShort"`
	Event             string `json:"event"`
	SourceBranch      string `json:"source"`
	DestinationBranch string `json:"destination"`
	CommitURL         string `json:"commitURL"`

	PullRequestContent `json:"pullRequestContent"`
	ProjectName        string     `json:"projectName"`
	Credential         Credential `json:"credential"`
	ExecutePath        string     `json:"ExecutePath"` //not need for codebuild
}

type CommentContent struct {
	Comment          string `json:"comment"`
	CommentAuthor    string `json:"commentAuthor"`
	PullRequestId    string `json:"pullRequestId"`
	PullRequestTitle string `json:"pullRequestTitle"`
}

type HookSender struct {
	Flavour           string `json:"flavour"`
	RepositoryName    string `json:"repositoryName"`
	RepositoryShort   string `json:"repositoryShort"`
	Event             string `json:"event"`
	SourceBranch      string `json:"source"`
	DestinationBranch string `json:"destination"`
	CommitURL         string `json:"commitURL"`

	CommentContent `json:"comment"`
}

type HookHandler struct {
	HandlerType string `json:"handlerType"`
	ProjectName string `json:"projectName"`
	ExecutePath string `json:"executePath"`
}

type HookEventV2 struct {
	HookSender  `json:"sender"`
	HookHandler `json:"handler"`
	Credential  `json:"credential"`
}

type HookSetter interface {
	Set(*events.APIGatewayProxyRequest, *HookEvent) error
	Match(*HookEvent, *QueueResult) bool
}
