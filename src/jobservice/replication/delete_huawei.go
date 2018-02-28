// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package replication

import (
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	//"github.com/vmware/harbor/src/common/utils/registry"
	//"github.com/vmware/harbor/src/common/utils/registry/auth"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"github.com/vmware/harbor/src/jobservice/config"
)

// Deleter deletes repository or tags
type HuaweiDeleter struct {
	repository string // prject_name/repo_name
	tags       []string

	dstURL string // url of target registry
	dstUsr string // username ...
	dstPwd string // username ...

	insecure bool

	//dstClient *registry.Repository

	logger *log.Logger
}

// NewDeleter returns a Deleter
func NewHuaweiDeleter(repository string, tags []string, dstURL, dstUsr, dstPwd string, insecure bool, logger *log.Logger) *HuaweiDeleter {
	deleter := &HuaweiDeleter{
		repository: repository,
		tags:       tags,
		dstURL:     dstURL,
		dstUsr:     dstUsr,
		dstPwd:     dstPwd,
		insecure:   insecure,
		logger:     logger,
	}
	deleter.logger.Infof("initialization completed: repository: %s, tags: %v, destination URL: %s, insecure: %v, destination user: %s",
		deleter.repository, deleter.tags, deleter.dstURL, deleter.insecure, deleter.dstUsr)
	return deleter
}

// Exit ...
func (d *HuaweiDeleter) Exit() error {
	return nil
}

// Enter deletes repository or tags
func (d *HuaweiDeleter) Enter() (string, error) {
	state, err := d.enter()
	if err != nil && retry(err) {
		d.logger.Info("waiting for retrying...")
		return models.JobRetrying, nil
	}

	return state, err
}
func  spliteRep(repo string)([]string){
	return strings.Split(repo, "/")
}
func (d *HuaweiDeleter) enter() (string, error) {
	nameSpace := spliteRep(d.repository)[0]
	repo      := spliteRep(d.repository)[1]
	url := strings.TrimRight(d.dstURL, "/") + "/dockyard/v2/domains/enncloud/namespaces/" + nameSpace + "/repositories/" + config.HUAWEI_PREFIX+repo + "/tags/"
	log.Debugf("huawei deleter has been called with tags:%v \n ",d.tags)
	// delete repository
	if len(d.tags) == 0 {
		u := url + "/tags"
		if err := del_huawei(u, d.dstUsr, d.dstPwd, d.insecure); err != nil {
			if err == errNotFound {
				d.logger.Warningf("repository %s does not exist on %s", d.repository, d.dstURL)
				return models.JobFinished, nil
			}
			d.logger.Errorf("an error occurred while deleting repository %s on %s with user %s: %v", d.repository, d.dstURL, d.dstUsr, err)
			return "", err

		}

		d.logger.Infof("repository %s on %s has been deleted", d.repository, d.dstURL)

		return models.JobFinished, nil

	}

	// delele tags
	for _, tag := range d.tags {
		u := url + tag
		if err := del_huawei(u, d.dstUsr, d.dstPwd, d.insecure); err != nil {
			if err == errNotFound {
				d.logger.Warningf("repository %s does not exist on %s", d.repository, d.dstURL)
				continue
			}

			d.logger.Errorf("an error occurred while deleting repository %s:%s on %s with user %s: %v", d.repository, tag, d.dstURL, d.dstUsr, err)
			return "", err
		}
		d.logger.Infof("repository %s:%s on %s has been deleted", d.repository, tag, d.dstURL)
	}
	return models.JobFinished, nil
}

func del_huawei(url, username, password string, insecure bool) error {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(username, password)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK  || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("%d %s", resp.StatusCode, string(b))
}
