package gitConnector

import (
	"encoding/base64"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
	"os"
	"strings"
)

var conf *oauth2.Config
var baseURL = "https://api.bitbucket.org/2.0/"
var buildKey []byte

type BitBucketConnector struct {
	HookEventPtr *hookEvent.HookEvent
}

func (connector *BitBucketConnector) initialize() {
	conf = &oauth2.Config{
		ClientID:     connector.HookEventPtr.Credential.Key,
		ClientSecret: connector.HookEventPtr.Credential.Secret,
		Scopes:       []string{"pullrequest", "repository", "pullrequest:write"}, //pullrequest to create tasks, pr:write to comment, repostory
		Endpoint:     bitbucket.Endpoint,
	}

	buildKey = []byte(strings.Split(os.Getenv("CODEBUILD_BUILD_ID"), ":")[1])
}
func (connector BitBucketConnector) Connect() {

}

func (connector BitBucketConnector) Comment(str string) {

}

func (connector BitBucketConnector) BuildStart() {

}

func (connector BitBucketConnector) BuildFail() {

}

func (connect BitBucketConnector) BuildStop() {

}

func (connector BitBucketConnector) UpdateBuildState(state string) {
	url := connector.HookEventPtr.CommitURL + "/statuses/build/" + base64.URLEncoding.EncodeToString(buildKey)
}
func (connector BitBucketConnector) BuildSucc() {

}
