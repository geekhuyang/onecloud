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

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseHostDriver struct {
}

func (self *SBaseHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SBaseHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewNotImplementedError("Not Implement ValidateAttachStorage")
}

func (self *SBaseHostDriver) RequestAttachStorage(ctx context.Context, hoststorage *models.SHoststorage, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("Not Implement RequestAttachStorage")
}

func (self *SBaseHostDriver) RequestDetachStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("Not Implement RequestDetachStorage")
}

func (self *SBaseHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return fmt.Errorf("Not Implement ValidateDiskSize")
}

func (self *SBaseHostDriver) RequestDeleteSnapshotsWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) RequestCleanUpDiskSnapshots(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) PrepareConvert(host *models.SHost, image, raid string, data jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Set("description", jsonutils.NewString("Baremetal convered Hypervisor"))
	name, err := data.Get("name")
	if err == nil {
		params.Set("name", name)
	} else {
		params.Set("name", jsonutils.NewString(host.Name))
	}
	params.Set("vcpu_count", jsonutils.NewInt(int64(host.CpuCount)))
	params.Set("vmem_size", jsonutils.NewInt(int64(host.MemSize)))
	params.Set("auto_start", jsonutils.NewBool(true))
	params.Set("is_system", jsonutils.NewBool(true))
	params.Set("prefer_baremetal", jsonutils.NewString(host.Id))
	params.Set("baremetal", jsonutils.NewBool(true))
	return params, nil
}

func (self *SBaseHostDriver) PrepareUnconvert(host *models.SHost) error {
	return nil
}

func (self *SBaseHostDriver) FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost) error {
	bss := host.GetBaremetalstorage()
	if bss != nil {
		bs := bss.GetStorage()
		if bs != nil {
			bs.SetStatus(userCred, models.STORAGE_ONLINE, "")
		} else {
			log.Errorf("ERROR: baremetal storage is None???")
		}
	} else {
		log.Errorf("ERROR: baremetal has no valid baremetalstorage????")
	}
	adminNetif := host.GetAdminNetInterface()
	if adminNetif == nil {
		return fmt.Errorf("admin netif is nil")
	}
	adminNic := adminNetif.GetBaremetalNetwork()
	if adminNic == nil {
		return fmt.Errorf("admin nic is nil")
	}
	db.Update(host, func() error {
		host.AccessIp = adminNic.IpAddr
		host.Enabled = true
		host.HostType = models.HOST_TYPE_BAREMETAL
		host.HostStatus = models.HOST_OFFLINE
		host.ManagerUri = ""
		host.Version = ""
		host.MemReserved = 0
		return nil
	})
	log.Infof("Do finish_unconvert!!!!!!!!")
	self.CleanSchedCache(host)
	return nil
}

func (self *SBaseHostDriver) CleanSchedCache(host *models.SHost) error {
	return host.ClearSchedDescCache()
}
func (self *SBaseHostDriver) FinishConvert(userCred mcclient.TokenCredential, host *models.SHost, guest *models.SGuest, hostType string) error {
	_, err := db.Update(guest, func() error {
		guest.VmemSize = 0
		guest.VcpuCount = 0
		return nil
	})
	if err != nil {
		return err
	}
	for _, guestdisk := range guest.GetDisks() {
		disk := guestdisk.GetDisk()
		db.Update(disk, func() error {
			disk.DiskSize = 0
			return nil
		})
	}
	bs := host.GetBaremetalstorage().GetStorage()
	bs.SetStatus(userCred, models.STORAGE_OFFLINE, "")
	db.Update(host, func() error {
		host.CpuReserved = 0
		host.MemReserved = 0
		host.AccessIp = guest.GetRealIPs()[0]
		host.Enabled = false
		host.HostStatus = models.HOST_OFFLINE
		host.HostType = hostType
		host.IsBaremetal = true
		return nil
	})
	self.CleanSchedCache(host)
	return nil
}

func (self *SBaseHostDriver) ConvertFailed(host *models.SHost) error {
	return self.CleanSchedCache(host)
}

func (self *SBaseHostDriver) GetRaidScheme(host *models.SHost, raid string) (string, error) {
	var candidates []string
	if len(raid) == 0 {
		candidates = []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1, baremetal.DISK_CONF_RAID5, baremetal.DISK_CONF_RAID0, baremetal.DISK_CONF_NONE}
	} else {
		if utils.IsInStringArray(raid, []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1}) {
			candidates = []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1}
		} else {
			candidates = []string{raid}
		}
	}
	var conf []*api.BaremetalDiskConfig
	for i := 0; i < len(candidates); i++ {
		if candidates[i] == baremetal.DISK_CONF_NONE {
			conf = []*api.BaremetalDiskConfig{}
		} else {
			parsedConf, err := baremetal.ParseDiskConfig(candidates[i])
			if err != nil {
				log.Errorf("try raid %s failed: %s", candidates[i], err.Error())
				return "", err
			}
			conf = []*api.BaremetalDiskConfig{&parsedConf}
		}
		baremetalStorage := models.ConvertStorageInfo2BaremetalStorages(host.StorageInfo)
		if baremetalStorage == nil {
			return "", fmt.Errorf("Convert storage info error")
		}
		layout, err := baremetal.CalculateLayout(conf, baremetalStorage)
		if err != nil {
			log.Errorf("try raid %s failed: %s", candidates[i], err.Error())
			continue
		}
		log.Infof("convert layout %v", layout)
		raid = candidates[i]
		break
	}
	if len(raid) == 0 {
		return "", fmt.Errorf("Disk misconfiguration")
	}
	return raid, nil
}

func (driver *SBaseHostDriver) IsReachStoragecacheCapacityLimit(host *models.SHost, cachedImages []models.SCachedimage) bool {
	return false
}

func (driver *SBaseHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return -1
}

func (driver *SBaseHostDriver) RequestDeallocateBackupDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}
