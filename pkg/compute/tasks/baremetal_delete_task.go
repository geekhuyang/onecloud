// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type BaremetalDeleteTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalDeleteTask{})
}

func (self *BaremetalDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_DELETE, "")
	if !baremetal.IsBaremetalAgentReady() {
		self.OnDeleteBaremetalComplete(ctx, baremetal, nil)
		return
	}
	url := fmt.Sprintf("/baremetals/%s/delete", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnDeleteBaremetalComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		if errVal, ok := err.(*httputils.JSONClientError); ok && errVal.Code == 404 {
			self.OnDeleteBaremetalComplete(ctx, baremetal, nil)
			return
		}
		log.Errorln(err.Error())
		self.OnFailure(ctx, baremetal, nil)
	}
}

func (self *BaremetalDeleteTask) OnDeleteBaremetalComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalDeleteTask) OnDeleteBaremetalCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.OnFailure(ctx, baremetal, body)
}

func (self *BaremetalDeleteTask) OnFailure(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_DELETE_FAIL, "")
	self.SetStageFailed(ctx, "")
}
