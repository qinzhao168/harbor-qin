package Driver

import (
	//"bytes"
	//"crypto/tls"
	//"encoding/json"
	//"fmt"
	//"io/ioutil"
	//"net/http"
	//"strings"
)
import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"github.com/vmware/harbor/src/jobservice/config"
)

type HuaweiRegisty struct {
	UserName  string
	Password  string
	Url       string
	Insecure  bool
}

func (reg *HuaweiRegisty) CreateProject(ProjectName string,public int) error  {
	project := struct {
		Namespace string `json:"namespace"`
	}{
		Namespace: config.HUAWEI_PREFIX + ProjectName,
	}

	data, err := json.Marshal(project)
	if err != nil {
		return err
	}

	url := strings.TrimRight(reg.Url, "/") + "/dockyard/v2/namespaces"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type","application/json;charset=utf8")
	req.Header.Set("X-Auth-Token",Generate())

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: reg.Insecure,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// version 0.1.1's reponse code is 200
	if resp.StatusCode == http.StatusCreated ||
		resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusConflict{
		return nil
	}

	//if resp.StatusCode == http.StatusConflict {
	//	return ErrConflict
	//}

	message, err := ioutil.ReadAll(resp.Body)


	return fmt.Errorf("failed to create project %s on %s with para %s: %d %s",
		ProjectName, url, string(data), resp.StatusCode, string(message))
	return  nil
}

