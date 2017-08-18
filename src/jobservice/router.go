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

package main

import (
	api "github.com/vmware/harbor/src/jobservice/api"

	"github.com/astaxie/beego"
)

func initRouters() {
	beego.Router("/api/jobs/replication", &api.ReplicationJob{})
	beego.Router("/api/jobs/replication/:id/log", &api.ReplicationJob{}, "get:GetLog")
	beego.Router("/api/jobs/replication/actions", &api.ReplicationJob{}, "post:HandleAction")
}
