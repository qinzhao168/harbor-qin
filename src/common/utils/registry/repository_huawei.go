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

package registry

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	//	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"

	"github.com/vmware/harbor/src/common/utils"
	registry_error "github.com/vmware/harbor/src/common/utils/registry/error"
	"github.com/vmware/harbor/src/jobservice/config"
)

// RepositoryHuawei holds information of a repository entity
type RepositoryHuawei struct {
	Name     string
	Endpoint *url.URL
	client   *http.Client
}

// NewRepositoryHuawei returns an instance of RepositoryHuawei
func NewRepositoryHuawei(name, endpoint string, client *http.Client) (*RepositoryHuawei, error) {
	name = strings.TrimSpace(name)

	u, err := utils.ParseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	repository := &RepositoryHuawei{
		Name:     config.HUAWEI_PREFIX + name,
		Endpoint: u,
		client:   client,
	}

	return repository, nil
}

// NewRepositoryHuaweiWithModifiers returns an instance of RepositoryHuawei according to the modifiers
func NewRepositoryHuaweiWithModifiers(name, endpoint string, insecure bool, modifiers ...Modifier) (RepositoryInterface, error) {

	transport := NewTransport(GetHTTPTransport(insecure), modifiers...)
	return NewRepositoryHuawei(name, endpoint, &http.Client{
		Transport: transport,
		//  for transferring large image, OS will handle i/o timeout
		//	Timeout:   30 * time.Second,
	})
}


// ListTag ...
func (r *RepositoryHuawei) ListTag() ([]string, error) {
	tags := []string{}
	req, err := http.NewRequest("GET", buildTagListURL(r.Endpoint.String(), r.Name), nil)
	if err != nil {
		return tags, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return tags, parseError(err)
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tags, err
	}

	if resp.StatusCode == http.StatusOK {
		tagsResp := struct {
			Tags []string `json:"tags"`
		}{}

		if err := json.Unmarshal(b, &tagsResp); err != nil {
			return tags, err
		}

		tags = tagsResp.Tags

		return tags, nil
	}
	return tags, &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}

}

// ManifestExist ...
func (r *RepositoryHuawei) ManifestExist(reference string) (digest string, exist bool, err error) {
	req, err := http.NewRequest("GET", buildManifestURL(r.Endpoint.String(), r.Name, reference), nil)
	if err != nil {
		return
	}

	req.Header.Add(http.CanonicalHeaderKey("Accept"), schema1.MediaTypeManifest)
	req.Header.Add(http.CanonicalHeaderKey("Accept"), schema2.MediaTypeManifest)

	resp, err := r.client.Do(req)
	if err != nil {
		err = parseError(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		exist = true
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
	return
}

// PullManifest ...
func (r *RepositoryHuawei) PullManifest(reference string, acceptMediaTypes []string) (digest, mediaType string, payload []byte, err error) {
	req, err := http.NewRequest("GET", buildManifestURL(r.Endpoint.String(), r.Name, reference), nil)
	if err != nil {
		return
	}

	for _, mediaType := range acceptMediaTypes {
		req.Header.Add(http.CanonicalHeaderKey("Accept"), mediaType)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		err = parseError(err)
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode == http.StatusOK {
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		mediaType = resp.Header.Get(http.CanonicalHeaderKey("Content-Type"))
		payload = b
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}

	return
}

// PushManifest ...
func (r *RepositoryHuawei) PushManifest(reference, mediaType string, payload []byte) (digest string, err error) {
	req, err := http.NewRequest("PUT", buildManifestURL(r.Endpoint.String(), r.Name, reference),
		bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set(http.CanonicalHeaderKey("Content-Type"), mediaType)

	resp, err := r.client.Do(req)
	if err != nil {
		err = parseError(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}

	return
}

// DeleteManifest ...
func (r *RepositoryHuawei) DeleteManifest(digest string) error {
	req, err := http.NewRequest("DELETE", buildManifestURL(r.Endpoint.String(), r.Name, digest), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return parseError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
}

// DeleteTag ...
func (r *RepositoryHuawei) DeleteTag(tag string) error {
	digest, exist, err := r.ManifestExist(tag)
	if err != nil {
		return err
	}

	if !exist {
		return &registry_error.Error{
			StatusCode: http.StatusNotFound,
		}
	}

	return r.DeleteManifest(digest)
}

// BlobExist ...
func (r *RepositoryHuawei) BlobExist(digest string) (bool, error) {
	req, err := http.NewRequest("GET", buildBlobURL(r.Endpoint.String(), r.Name, digest), nil)
	if err != nil {
		return false, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return false, parseError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return false, &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
}

// PullBlob : client must close data if it is not nil
func (r *RepositoryHuawei) PullBlob(digest string) (size int64, data io.ReadCloser, err error) {
	req, err := http.NewRequest("GET", buildBlobURL(r.Endpoint.String(), r.Name, digest), nil)
	if err != nil {
		return
	}

	resp, err := r.client.Do(req)
	if err != nil {
		err = parseError(err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		contengLength := resp.Header.Get(http.CanonicalHeaderKey("Content-Length"))
		size, err = strconv.ParseInt(contengLength, 10, 64)
		if err != nil {
			return
		}
		data = resp.Body
		return
	}
	// can not close the connect if the status code is 200
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}

	return
}

func (r *RepositoryHuawei)patchUploads(uuid string, data io.Reader) (location, uploadUUID string, err error) {
	req, err := http.NewRequest("PATCH", buildInitiateBlobUploadWithUuidURL(r.Endpoint.String(), r.Name, uuid), data)


	resp, err := r.client.Do(req)
	if nil != err {
		err = parseError(err)
		return
	}
	defer resp.Body.Close()

	if http.StatusCreated == resp.StatusCode || http.StatusAccepted == resp.StatusCode{
		location = resp.Header.Get(http.CanonicalHeaderKey("Location"))
		uploadUUID = resp.Header.Get(http.CanonicalHeaderKey("Docker-Upload-Uuid"))
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
	return
}

func (r *RepositoryHuawei)putUploads( digest, uuid string) (err error) {
	req, err := http.NewRequest("PUT", buildBlobUploadURL(r.Endpoint.String(), r.Name, uuid, digest), nil)

	resp, err := r.client.Do(req)
	if nil != err {
		err = parseError(err)
		return
	}

	defer resp.Body.Close()

	if http.StatusAccepted == resp.StatusCode || http.StatusCreated == resp.StatusCode {
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
	return
}

func (r *RepositoryHuawei) initiateBlobUpload(name string) (location, uploadUUID string, err error) {
	req, err := http.NewRequest("POST", buildInitiateBlobUploadURL(r.Endpoint.String(), r.Name), nil)
	req.Header.Set(http.CanonicalHeaderKey("Content-Length"), "0")

	resp, err := r.client.Do(req)
	if err != nil {
		err = parseError(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		location = resp.Header.Get(http.CanonicalHeaderKey("Location"))
		uploadUUID = resp.Header.Get(http.CanonicalHeaderKey("Docker-Upload-UUID"))
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}

	return
}

func (r *RepositoryHuawei) monolithicBlobUpload(location, digest string, size int64, data io.Reader) error {
	req, err := http.NewRequest("PUT", buildMonolithicBlobUploadURL(location, digest), data)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return parseError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
}

// PushBlob ...
func (r *RepositoryHuawei) PushBlob(digest string, size int64, data io.Reader) error {
	_, uuid, err := r.initiateBlobUpload(r.Name)
	if err != nil {
		return err
	}
	_,_,err = r.patchUploads(uuid, data)
	if err != nil {
		return err
	}
	return r.putUploads(digest, uuid)
}

// DeleteBlob ...
func (r *RepositoryHuawei) DeleteBlob(digest string) error {
	req, err := http.NewRequest("DELETE", buildBlobURL(r.Endpoint.String(), r.Name, digest), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return parseError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return &registry_error.Error{
		StatusCode: resp.StatusCode,
		Detail:     string(b),
	}
}
