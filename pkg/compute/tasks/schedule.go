package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	SCHEDULE        = models.VM_SCHEDULE
	SCHEDULE_FAILED = models.VM_SCHEDULE_FAILED
)

type IScheduleModel interface {
	db.IStandaloneModel

	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
}

type IScheduleTask interface {
	GetUserCred() mcclient.TokenCredential
	GetParams() *jsonutils.JSONDict
	GetPendingUsage(quota quotas.IQuota) error

	SetStage(stageName string, data *jsonutils.JSONDict)
	SetStageFailed(ctx context.Context, reason string)
	OnScheduleFailCallback(obj IScheduleModel)
	OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict)
	SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string)
}

func StartScheduleObjects(
	ctx context.Context,
	task IScheduleTask,
	objs []db.IStandaloneModel,
) {
	schedObjs := make([]IScheduleModel, len(objs))
	for i, obj := range objs {
		schedObj := obj.(IScheduleModel)
		schedObjs[i] = schedObj
		db.OpsLog.LogEvent(schedObj, db.ACT_ALLOCATING, nil, task.GetUserCred())
		schedObj.SetStatus(task.GetUserCred(), SCHEDULE, "")
	}
	doScheduleObjects(ctx, task, schedObjs)
}

func doScheduleObjects(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
) {
	schedtags := models.ApplySchedPolicies(task.GetParams())

	task.SetStage("OnScheduleComplete", schedtags)

	s := auth.GetAdminSession(options.Options.Region, "")
	results, err := modules.SchedManager.DoSchedule(s, task.GetParams(), len(objs))
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, fmt.Sprintf("Scheduler fail: %s", err))
		return
	}
	onSchedulerResults(ctx, task, objs, results)
}

func cancelPendingUsage(ctx context.Context, task IScheduleTask) {
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("Taks GetPendingUsage fail %s", err)
		return
	}
	ownerProjectId, _ := task.GetParams().GetString("owner_tenant_id")
	err = models.QuotaManager.CancelPendingUsage(ctx, task.GetUserCred(), ownerProjectId, &pendingUsage, &pendingUsage)
	if err != nil {
		log.Errorf("cancelpendingusage error %s", err)
	}
}

func onSchedulerRequestFail(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	reason string,
) {
	for _, obj := range objs {
		onScheduleFail(ctx, task, obj, reason)
	}
	task.SetStageFailed(ctx, fmt.Sprintf("Schedule failed: %s", reason))
	cancelPendingUsage(ctx, task)
}

func onScheduleFail(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	msg string,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	reason := "No matching resources"
	if len(msg) > 0 {
		reason = fmt.Sprintf("%s: %s", reason, msg)
	}

	obj.SetStatus(task.GetUserCred(), SCHEDULE_FAILED, reason)
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATE_FAIL, reason, task.GetUserCred())
	notifyclient.NotifySystemError(obj.GetId(), obj.GetName(), SCHEDULE_FAILED, reason)
	task.OnScheduleFailCallback(obj)
}

func onSchedulerResults(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	results []jsonutils.JSONObject,
) {
	succCount := 0
	for idx := 0; idx < len(objs); idx += 1 {
		obj := objs[idx]
		result := results[idx]
		if result.Contains("candidate") {
			hostId, _ := result.GetString("candidate", "id")
			onScheduleSucc(ctx, task, obj, hostId)
			succCount += 1
		} else if result.Contains("error") {
			msg, _ := result.Get("error")
			onScheduleFail(ctx, task, obj, fmt.Sprintf("%s", msg))
		} else {
			msg := fmt.Sprintf("Unknown scheduler result %s", result)
			onScheduleFail(ctx, task, obj, msg)
			return
		}
	}
	if succCount == 0 {
		task.SetStageFailed(ctx, "Schedule failed")
	}
	cancelPendingUsage(ctx, task)
}

func onScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	hostId string,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	task.SaveScheduleResult(ctx, obj, hostId)
	models.HostManager.ClearSchedDescCache(hostId)
}