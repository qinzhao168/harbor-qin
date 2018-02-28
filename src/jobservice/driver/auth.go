package Driver

import (
"encoding/json"
"io/ioutil"
"net/http"
"strings"
"sync"
"time"

"github.com/golang/glog"
)

var (
	iam      = "https://iam.cn-north-1.myhwclouds.com/v3/auth/tokens"
	authInfo = `{"auth":{"identity":{"methods":["password"],"password":{"user":{"name":"enncloud","password":"enN12345","domain":{"name":"enncloud"}}}},"scope":{"project":{"name":"cn-north-1"}}}}`
	token    = &Authorizer{}
)

func init() {
	getFromHuawei()
}

// Authorizer store the token info
type Authorizer struct {
	Value     string     `json:"-"`
	ExpiresAt *time.Time `json:"expires_at"`
	sync.Mutex
}

// Generate Generate token string, if the token was expired ,will update the token
func Generate() string {
	if time.Now().Sub(*token.ExpiresAt) < 0 {
		return token.Value
	}

	freshToken, err := getFromHuawei()
	if err != nil {
		return ""
	}
	token = freshToken
	return token.Value
}

// getFromHuawei get huawei's athorization token when the init func is call or the token is expired
func getFromHuawei() (*Authorizer, error) {
	response, err := http.Post(iam, "application/json", strings.NewReader(authInfo))
	if err != nil {
		glog.Errorf("init thirty party plateform tokens err: %v", err)
		return nil, err
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		glog.Errorf("read the thirty party plateform tokens response err: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	translate := &struct {
		*Authorizer `json:"token"`
	}{}
	if err = json.Unmarshal(contents, translate); err != nil {
		glog.Errorf("json unmarshal thirty party plateform tokens response err: %v", err)
		return nil, err
	}
	token = translate.Authorizer
	token.Value = response.Header.Get("X-Subject-Token")
	return translate.Authorizer, nil
}