package hookEvent

import "github.com/aws/aws-lambda-go/events"

type Credential struct {
	Category string `json:"category"` // bitbucket:oauth
	Key      string `json:"key"`
	Secret   string `json:"secret"`
}

type HookEvent struct {
	GitFlavour        string     `json:"gitFlavour"`
	RepositoryName    string     `json:"repositoryName"`
	RepositoryShort   string     `json:"repositoryShort"`
	ProjectName       string     `json:"projectName"`
	Event             string     `json:"event"`
	SourceBranch      string     `json:"source"`
	DestinationBranch string     `json:"destination"`
	Comment           string     `json:"comment"`
	CommentAuthor     string     `json:"commentAuthor"`
	Uuid              string     `json:"uuid"`
	CommitURL         string     `json:"commitURL"`
	PullRequestId     string     `json:"pullRequestId"`
	Credential        Credential `json:"credential"`
	ExecutePath       string     `json:"ExecutePath"`
}

type HookSetter interface {
	Set(*events.APIGatewayProxyRequest, *HookEvent) error
}
