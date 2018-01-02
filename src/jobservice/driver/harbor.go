package Driver

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type HarborRegisty struct {
	UserName  string
	Password  string
	Url       string
	Insecure  bool
}

func (reg *HarborRegisty) CreateProject(ProjectName string,public int) error  {
	project := struct {
		ProjectName string `json:"project_name"`
		Public      int    `json:"public"`
	}{
		ProjectName: ProjectName,
		Public:      public,
	}

	data, err := json.Marshal(project)
	if err != nil {
		return err
	}

	url := strings.TrimRight(reg.Url, "/") + "/api/projects/"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.SetBasicAuth(reg.UserName, reg.Password)

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
		resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusConflict {
		return ErrConflict
	}

	message, err := ioutil.ReadAll(resp.Body)


	return fmt.Errorf("failed to create project %s on %s with user %s: %d %s",
		ProjectName, reg.Url, reg.UserName, resp.StatusCode, string(message))
}
