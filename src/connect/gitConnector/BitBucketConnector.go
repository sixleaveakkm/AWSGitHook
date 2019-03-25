package gitConnector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sixleaveakkm/AWSGitHook/src/hookEvent"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

var buildKey string
var token string

type BitBucketConnector struct {
	HookEventPtr *hookEvent.HookEvent
}

func (connector *BitBucketConnector) Initialize() {
	buildKey = os.Getenv("CODEBUILD_BUILD_ID")
}
func (connector BitBucketConnector) Connect() {
	fmt.Printf("connecting to bitbucket...")
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://bitbucket.org/site/oauth2/access_token", strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		fmt.Printf("Error creating http post, %v", err)
	}
	req.SetBasicAuth(connector.HookEventPtr.Credential.Key, connector.HookEventPtr.Credential.Secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error execute http post, %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error close http response, %v", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error read body, %v", err)
	}

	tokenResponse := new(TokenResponse)
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Printf("Error parse json, %v", err)
	}
	token = tokenResponse.AccessToken
	fmt.Printf("Get token succ")
}

type CommentContent struct {
	Raw string `json:"raw"`
}
type CommentFormat struct {
	Content CommentContent `json:"content"`
}

func (connector BitBucketConnector) Comment(str string) {
	connector.Connect()
	fmt.Printf("Post comment: %s", str)
	commentUrl := "https://api.bitbucket.org/2.0/repositories/" + connector.HookEventPtr.RepositoryShort + "/pullrequests/" + connector.HookEventPtr.PullRequestId + "/comments"
	client := &http.Client{}
	comment := CommentFormat{
		Content: CommentContent{
			Raw: str,
		},
	}
	jsonBytes, err := json.Marshal(comment)
	if err != nil {
		fmt.Printf("Error when format comment json byte, %v", err)
	}
	req, err := http.NewRequest("POST", commentUrl, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Printf("Error creating http post, %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error execute http post, %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error close http response, %v", err)
		}
	}()
	if resp.StatusCode == 201 {
		fmt.Printf("Add comment success")
	}

}

func (connector BitBucketConnector) BuildStart() {
	connector.UpdateBuildState("INPROGRESS")
}

func (connector BitBucketConnector) BuildFail() {
	connector.UpdateBuildState("FAILED")
}

func (connector BitBucketConnector) BuildStop() {
	connector.UpdateBuildState("STOPPED")
}

func (connector BitBucketConnector) UpdateBuildState(state string) {
	connector.Connect()
	fmt.Printf("update build state... %s", state)
	buildStateUrl := connector.HookEventPtr.CommitURL + "/statuses/build"
	client := &http.Client{}
	form := url.Values{}
	form.Add("state", state)
	form.Add("url", "https://console.aws.amazon.com/codesuite/codebuild/projects/"+connector.HookEventPtr.ProjectName+"/build/"+buildKey+"/log")
	form.Add("key", "DEPLOY-1") //todo: key is used for separate different build, auto assign
	req, err := http.NewRequest("POST", buildStateUrl, strings.NewReader(form.Encode()))
	if err != nil {
		fmt.Printf("Error creating http post, %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error execute http post, %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error close http response, %v", err)
		}
	}()
	if resp.StatusCode == 201 {
		fmt.Printf("Execute update build success")
	}

}

func (connector BitBucketConnector) BuildSucc() {
	connector.UpdateBuildState("SUCCESSFUL")
}

func (connector BitBucketConnector) GetToken() string {
	return token
}
