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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DiskBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(DiskBatchCreateTask{})
}

func (self *DiskBatchCreateTask) getNeedScheduleDisks(objs []db.IStandaloneModel) []db.IStandaloneModel {
	toSchedDisks := make([]db.IStandaloneModel, 0)
	for _, obj := range objs {
		disk := obj.(*models.SDisk)
		if disk.StorageId == "" {
			toSchedDisks = append(toSchedDisks, disk)
		}
	}
	return toSchedDisks
}

func (self *DiskBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	toSchedDisks := self.getNeedScheduleDisks(objs)
	if len(toSchedDisks) == 0 {
		self.SetStage("OnScheduleComplete", nil)
		// create not need schedule disks directly
		for _, disk := range objs {
			self.startCreateDisk(ctx, disk.(*models.SDisk))
		}
		return
	}
	StartScheduleObjects(ctx, self, toSchedDisks)
}

func (self *DiskBatchCreateTask) GetCreateInput() (*api.DiskCreateInput, error) {
	input := new(api.DiskCreateInput)
	err := self.GetParams().Unmarshal(input)
	return input, err
}

func (self *DiskBatchCreateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	input, err := self.GetCreateInput()
	if err != nil {
		return nil, err
	}
	ret := new(schedapi.ScheduleInput)
	srvInput := input.ToServerCreateInput()
	err = srvInput.JSON(srvInput).Unmarshal(ret)
	return ret, err
}

func (self *DiskBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	self.SSchedTask.OnScheduleFailCallback(ctx, obj, reason)
	disk := obj.(*models.SDisk)
	log.Errorf("Schedule disk %s failed", disk.Name)
}

func (self *DiskBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource) {
	var err error
	disk := obj.(*models.SDisk)
	pendingUsage := models.SQuota{}
	err = self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}

	quotaStorage := models.SQuota{Storage: disk.DiskSize}

	onError := func(err error) {
		models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &quotaStorage)
		disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(disk.Id, disk.Name, models.DISK_ALLOC_FAILED, err.Error())
	}

	diskConfig, err := self.GetFirstDisk()
	if err != nil {
		onError(err)
		return
	}

	//err = disk.SetStorageByHost(hostId, &diskConfig)
	err = disk.SetStorage(candidate.Disks[0].StorageId, diskConfig)
	if err != nil {
		models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &quotaStorage)
		disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(disk.Id, disk.Name, models.DISK_ALLOC_FAILED, err.Error())
		return
	}

	self.startCreateDisk(ctx, disk)
}

func (self *DiskBatchCreateTask) startCreateDisk(ctx context.Context, disk *models.SDisk) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	quotaStorage := models.SQuota{Storage: disk.DiskSize}
	models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &quotaStorage)
	self.SetPendingUsage(&pendingUsage)

	disk.StartDiskCreateTask(ctx, self.GetUserCred(), false, "", self.GetTaskId())
}

func (self *DiskBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
