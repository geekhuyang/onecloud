package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

const (
	// PENDING：表示创建中
	// LAUNCH_FAILED：表示创建失败
	// RUNNING：表示运行中
	// STOPPED：表示关机
	// STARTING：表示开机中
	// STOPPING：表示关机中
	// REBOOTING：表示重启中
	// SHUTDOWN：表示停止待销毁
	// TERMINATING：表示销毁中。

	InstanceStatusStopped  = "STOPPED"
	InstanceStatusRunning  = "RUNNING"
	InstanceStatusStopping = "STOPPING"
	InstanceStatusStarting = "STARTING"
)

type SystemDisk struct {
	DiskType string  //系统盘类型。系统盘类型限制详见CVM实例配置。取值范围：LOCAL_BASIC：本地硬盘 LOCAL_SSD：本地SSD硬盘 CLOUD_BASIC：普通云硬盘 CLOUD_SSD：SSD云硬盘 CLOUD_PREMIUM：高性能云硬盘 默认取值：CLOUD_BASIC。
	DiskId   string  //	系统盘ID。LOCAL_BASIC 和 LOCAL_SSD 类型没有ID。暂时不支持该参数。
	DiskSize float32 //系统盘大小，单位：GB。默认值为 50
}

type DataDisk struct {
	DiskSize           float32 //	数据盘大小，单位：GB。最小调整步长为10G，不同数据盘类型取值范围不同，具体限制详见：CVM实例配置。默认值为0，表示不购买数据盘。更多限制详见产品文档。
	DiskType           string  //	数据盘类型。数据盘类型限制详见CVM实例配置。取值范围：LOCAL_BASIC：本地硬盘 LOCAL_SSD：本地SSD硬盘 CLOUD_BASIC：普通云硬盘 CLOUD_PREMIUM：高性能云硬盘 CLOUD_SSD：SSD云硬盘 默认取值：LOCAL_BASIC。 该参数对ResizeInstanceDisk接口无效。
	DiskId             string  //	数据盘ID。LOCAL_BASIC 和 LOCAL_SSD 类型没有ID。暂时不支持该参数。
	DeleteWithInstance bool    //	数据盘是否随子机销毁。取值范围：TRUE：子机销毁时，销毁数据盘 FALSE：子机销毁时，保留数据盘 默认取值：TRUE 该参数目前仅用于 RunInstances 接口。
}

type InternetAccessible struct {
	InternetChargeType      string //网络计费类型。取值范围：BANDWIDTH_PREPAID：预付费按带宽结算 TRAFFIC_POSTPAID_BY_HOUR：流量按小时后付费 BANDWIDTH_POSTPAID_BY_HOUR：带宽按小时后付费 BANDWIDTH_PACKAGE：带宽包用户 默认取值：非带宽包用户默认与子机付费类型保持一致。
	InternetMaxBandwidthOut int    //	公网出带宽上限，单位：Mbps。默认值：0Mbps。不同机型带宽上限范围不一致，具体限制详见购买网络带宽。
	PublicIpAssigned        bool   //	是否分配公网IP。取值范围: TRUE：表示分配公网IP FALSE：表示不分配公网IP 当公网带宽大于0Mbps时，可自由选择开通与否，默认开通公网IP；当公网带宽为0，则不允许分配公网IP。
}

type VirtualPrivateCloud struct {
	VpcId              string   //	私有网络ID，形如vpc-xxx。有效的VpcId可通过登录控制台查询；也可以调用接口 DescribeVpcEx ，从接口返回中的unVpcId字段获取。
	SubnetId           string   //	私有网络子网ID，形如subnet-xxx。有效的私有网络子网ID可通过登录控制台查询；也可以调用接口 DescribeSubnets ，从接口返回中的unSubnetId字段获取。
	AsVpcGateway       bool     //	是否用作公网网关。公网网关只有在实例拥有公网IP以及处于私有网络下时才能正常使用。取值范围：TRUE：表示用作公网网关 FALSE：表示不用作公网网关 默认取值：FALSE。
	PrivateIpAddresses []string //	私有网络子网 IP 数组，在创建实例、修改实例vpc属性操作中可使用此参数。当前仅批量创建多台实例时支持传入相同子网的多个 IP。
}

type LoginSettings struct {
	Password       string   //实例登录密码。不同操作系统类型密码复杂度限制不一样，具体如下：Linux实例密码必须8到16位，至少包括两项[a-z，A-Z]、[0-9] 和 [( ) ~ ! @ # $ % ^ & * - + = &#124; { } [ ] : ; ' , . ? / ]中的特殊符号。<br><li>Windows实例密码必须12到16位，至少包括三项[a-z]，[A-Z]，[0-9] 和 [( ) ~ ! @ # $ % ^ & * - + = { } [ ] : ; ' , . ? /]中的特殊符号。 若不指定该参数，则由系统随机生成密码，并通过站内信方式通知到用户。
	KeyIds         []string //	密钥ID列表。关联密钥后，就可以通过对应的私钥来访问实例；KeyId可通过接口DescribeKeyPairs获取，密钥与密码不能同时指定，同时Windows操作系统不支持指定密钥。当前仅支持购买的时候指定一个密钥。
	KeepImageLogin string   //	保持镜像的原始设置。该参数与Password或KeyIds.N不能同时指定。只有使用自定义镜像、共享镜像或外部导入镜像创建实例时才能指定该参数为TRUE。取值范围: TRUE：表示保持镜像的登录设置 FALSE：表示不保持镜像的登录设置 默认取值：FALSE。
}

type Tag struct {
	Key   string
	Value string
}

type SInstance struct {
	host *SHost

	image  *SImage
	idisks []cloudprovider.ICloudDisk

	Placement           Placement
	InstanceId          string
	InstanceType        string
	CPU                 int8
	Memory              int
	RestrictState       string //NORMAL EXPIRED PROTECTIVELY_ISOLATED
	InstanceName        string
	InstanceChargeType  InstanceChargeType  //PREPAID：表示预付费，即包年包月 POSTPAID_BY_HOUR：表示后付费，即按量计费 CDHPAID：CDH付费，即只对CDH计费，不对CDH上的实例计费。
	SystemDisk          SystemDisk          //实例系统盘信息。
	DataDisks           []DataDisk          //实例数据盘信息。只包含随实例购买的数据盘。
	PrivateIpAddresses  []string            //实例主网卡的内网IP列表。
	PublicIpAddresses   []string            //实例主网卡的公网IP列表。
	InternetAccessible  InternetAccessible  //实例带宽信息。
	VirtualPrivateCloud VirtualPrivateCloud //实例所属虚拟私有网络信息。
	ImageId             string              //	生产实例所使用的镜像ID。
	RenewFlag           string              //	自动续费标识。取值范围：NOTIFY_AND_MANUAL_RENEW：表示通知即将过期，但不自动续费 NOTIFY_AND_AUTO_RENEW：表示通知即将过期，而且自动续费 DISABLE_NOTIFY_AND_MANUAL_RENEW：表示不通知即将过期，也不自动续费。
	CreatedTime         time.Time           //	创建时间。按照ISO8601标准表示，并且使用UTC时间。格式为：YYYY-MM-DDThh:mm:ssZ。
	ExpiredTime         time.Time           //	到期时间。按照ISO8601标准表示，并且使用UTC时间。格式为：YYYY-MM-DDThh:mm:ssZ。
	OsName              string              //	操作系统名称。
	SecurityGroupIds    []string            //	实例所属安全组。该参数可以通过调用 DescribeSecurityGroups 的返回值中的sgId字段来获取。
	LoginSettings       LoginSettings       //实例登录设置。目前只返回实例所关联的密钥。
	InstanceState       string              //	实例状态。取值范围：PENDING：表示创建中 LAUNCH_FAILED：表示创建失败 RUNNING：表示运行中 STOPPED：表示关机 STARTING：表示开机中 STOPPING：表示关机中 REBOOTING：表示重启中 SHUTDOWN：表示停止待销毁 TERMINATING：表示销毁中。
	Tags                []Tag
}

func (self *SRegion) GetInstances(zoneId string, ids []string, offset int, limit int) ([]SInstance, int, error) {
	params := make(map[string]string)
	if limit < 1 || limit > 50 {
		limit = 50
	}

	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	instances := make([]SInstance, 0)
	if ids != nil && len(ids) > 0 {
		for index, id := range ids {
			params[fmt.Sprintf("InstanceIds.%d", index)] = id
		}
	} else {
		if len(zoneId) > 0 {
			params["Filters.0.Name"] = "zone"
			params["Filters.0.Values.0"] = zoneId
		}
	}
	body, err := self.cvmRequest("DescribeInstances", params)
	if err != nil {
		return nil, 0, err
	}
	err = body.Unmarshal(&instances, "InstanceSet")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return instances, int(total), nil
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	if self.image == nil {
		image, err := self.host.zone.region.GetImage(self.ImageId)
		if err == nil {
			self.image = image
		}
	}

	if self.image != nil {
		data.Add(jsonutils.NewString(self.image.OsName), "os_distribution")
	}

	return data
}

func (self *SInstance) GetCreateTime() time.Time {
	return self.CreatedTime
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	return self.InstanceName
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VirtualPrivateCloud.VpcId)
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, total, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, 50)
	if err != nil {
		log.Errorf("fetchDisks fail %s", err)
		return nil, err
	}
	if total > len(disks) {
		disks, _, err = self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, total)
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := self.host.zone.getStorageByCategory(disks[i].DiskType)
		if err != nil {
			return nil, err
		}
		disks[i].storage = store
		idisks[i] = &disks[i]
	}
	if utils.IsInStringArray(self.SystemDisk.DiskType, []string{"LOCAL_BASIC", "LOCAL_SSD"}) {
		storage := SLocalStorage{zone: self.host.zone, storageType: self.SystemDisk.DiskType}
		disk := SLocalDisk{
			storage:   &storage,
			DiskId:    self.SystemDisk.DiskId,
			DiskSize:  self.SystemDisk.DiskSize,
			DisktType: self.SystemDisk.DiskType,
			DiskUsage: "SYSTEM_DISK",
		}
		idisks = append(idisks, &disk)
	}

	for i := 0; i < len(self.DataDisks); i++ {
		if utils.IsInStringArray(self.DataDisks[i].DiskType, []string{"LOCAL_BASIC", "LOCAL_SSD"}) {
			storage := SLocalStorage{zone: self.host.zone, storageType: self.DataDisks[i].DiskType}
			disk := SLocalDisk{
				storage:   &storage,
				DiskId:    self.DataDisks[i].DiskId,
				DiskSize:  self.DataDisks[i].DiskSize,
				DisktType: self.DataDisks[i].DiskType,
				DiskUsage: "DATA_DISK",
			}
			idisks = append(idisks, &disk)
		}
	}

	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)
	for _, ip := range self.VirtualPrivateCloud.PrivateIpAddresses {
		nic := SInstanceNic{instance: self, ipAddr: ip}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (self *SInstance) GetVcpuCount() int8 {
	return self.CPU
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.Memory * 1024
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) GetOSType() string {
	if self.image == nil {
		image, err := self.host.zone.region.GetImage(self.ImageId)
		if err != nil {
			return self.OsName
		}
		self.image = image
	}
	if self.image != nil {
		switch self.image.Platform {
		case "Windows":
			return "Windows"
		case "CentOS", "Debian", "FreeBSD", "SUSE", "openSUSE":
			return "Linux"
		default:
			return "Linux"
		}
	}
	return self.OsName
}

func (self *SInstance) GetOSName() string {
	return self.OsName
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetStatus() string {
	switch self.InstanceState {
	case "PENDING":
		return models.VM_DEPLOYING
	case "LAUNCH_FAILED":
		return models.VM_DEPLOY_FAILED
	case "RUNNING":
		return models.VM_RUNNING
	case "STOPPED":
		return models.VM_READY
	case "STARTING", "REBOOTING":
		return models.VM_STARTING
	case "STOPPING":
		return models.VM_STOPPING
	case "SHUTDOWN":
		return models.VM_DEALLOCATED
	case "TERMINATING":
		return models.VM_DELETING
	default:
		return models.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	new, err := self.host.zone.region.GetInstance(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_QCLOUD
}

func (self *SInstance) StartVM() error {
	timeout := 300 * time.Second
	interval := 15 * time.Second

	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := self.Refresh()
		if err != nil {
			return err
		}
		log.Debugf("status %s expect %s", self.GetStatus(), models.VM_RUNNING)
		if self.GetStatus() == models.VM_RUNNING {
			return nil
		}
		if self.GetStatus() == models.VM_READY {
			err := self.host.zone.region.StartVM(self.InstanceId)
			if err != nil {
				return err
			}
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) StopVM(isForce bool) error {
	err := self.host.zone.region.StopVM(self.InstanceId, isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	url, err := self.host.zone.region.GetInstanceVNCUrl(self.InstanceId)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString("https://img.qcloud.com/qcloud/app/active_vnc/index.html?InstanceVncUrl="+url), "url")
	ret.Add(jsonutils.NewString("qcloud"), "protocol")
	ret.Add(jsonutils.NewString(self.InstanceId), "instance_id")
	return ret, nil
}

func (self *SInstance) UpdateVM(name string) error {
	return self.host.zone.region.UpdateVM(self.InstanceId, name)
}

func (self *SInstance) DeployVM(name string, password string, publicKey string, deleteKeypair bool, description string) error {
	var keypairName string
	if len(publicKey) > 0 {
		var err error
		keypairName, err = self.host.zone.region.syncKeypair(publicKey)
		if err != nil {
			return err
		}
	}

	return self.host.zone.region.DeployVM(self.InstanceId, name, password, keypairName, deleteKeypair, description)
}

func (self *SInstance) RebuildRoot(imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	keypair := ""
	if len(publicKey) > 0 {
		var err error
		keypair, err = self.host.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}
	err := self.host.zone.region.ReplaceSystemDisk(self.InstanceId, imageId, passwd, keypair, sysSizeGB)
	if err != nil {
		return "", err
	}
	self.StopVM(true)
	instance, err := self.host.zone.region.GetInstance(self.InstanceId)
	if err != nil {
		return "", err
	}
	return instance.SystemDisk.DiskId, nil
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	return self.host.zone.region.ChangeVMConfig(self.Placement.Zone, self.InstanceId, ncpu, vmem, nil)
}

func (self *SInstance) AttachDisk(diskId string) error {
	return self.host.zone.region.AttachDisk(self.InstanceId, diskId)
}

func (self *SInstance) DetachDisk(diskId string) error {
	return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, _, err := self.GetInstances("", []string{instanceId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &instances[0], nil
}

func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, networkId string, ipAddr string,
	keypair string, userData string) (string, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["ImageId"] = imageId
	params["InstanceType"] = instanceType
	//params["SecurityGroupId"] = securityGroupId
	params["Placement.Zone"] = zoneId
	params["InstanceName"] = name
	params["Description"] = desc
	params["InstanceChargeType"] = "POSTPAID_BY_HOUR"
	params["InternetAccessible.InternetMaxBandwidthOut"] = "100"
	params["InternetAccessible.PublicIpAssigned"] = "FALSE"
	params["HostName"] = name
	if len(passwd) > 0 {
		params["LoginSettings.Password"] = passwd
	} else {
		params["LoginSettings.KeepImageLogin"] = "TRUE"
	}

	if len(userData) > 0 {
		params["UserData"] = userData
	}

	//params["IoOptimized"] = "optimized"
	for i, d := range disks {
		if i == 0 {
			params["SystemDisk.DiskType"] = d.DiskType
			params["SystemDisk.DiskSize"] = fmt.Sprintf("%d", d.DiskSize)
		} else {
			params[fmt.Sprintf("DataDisks.%d.DiskSize", i-1)] = fmt.Sprintf("%d", d.DiskSize)
			params[fmt.Sprintf("DataDisks.%d.DiskType", i-1)] = d.DiskType
		}
	}
	network, err := self.GetNetwork(networkId)
	if err != nil {
		return "", err
	}
	params["VirtualPrivateCloud.SubnetId"] = networkId
	params["VirtualPrivateCloud.VpcId"] = network.VpcId
	if len(ipAddr) > 0 {
		params["VirtualPrivateCloud.PrivateIpAddresses.0"] = ipAddr
	}
	// if len(keypair) > 0 {
	// 	params["KeyPairName"] = keypair
	// }
	params["ClientToken"] = utils.GenRequestId(20)
	//log.Errorf("create params: %s", jsonutils.Marshal(params).PrettyString())
	instanceIdSet := []string{}
	body, err := self.cvmRequest("RunInstances", params)
	if err != nil {
		log.Errorf("RunInstances fail %s", err)
		return "", err
	}
	err = body.Unmarshal(&instanceIdSet, "InstanceIdSet")
	if err == nil && len(instanceIdSet) > 0 {
		return instanceIdSet[0], nil
	}
	return "", fmt.Errorf("Failed to create instance")
}

func (self *SRegion) doStartVM(instanceId string) error {
	return self.instanceOperation(instanceId, "StartInstances", nil)
}

func (self *SRegion) doStopVM(instanceId string, isForce bool) error {
	params := make(map[string]string)
	if isForce {
		params["ForceStop"] = "true"
	} else {
		params["ForceStop"] = "false"
	}
	return self.instanceOperation(instanceId, "StopInstances", params)
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	params := make(map[string]string)
	return self.instanceOperation(instanceId, "TerminateInstances", params)
}

func (self *SRegion) StartVM(instanceId string) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StartVM: %s", err)
		return err
	}
	if status != InstanceStatusStopped {
		log.Errorf("StartVM: vm status is %s expect %s", status, InstanceStatusStopped)
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStartVM(instanceId)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StopVM: %s", err)
		return err
	}
	if status == InstanceStatusStopped {
		return nil
	}
	return self.doStopVM(instanceId, isForce)
}

func (self *SRegion) DeleteVM(instanceId string) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on DeleteVM: %s", err)
		return err
	}
	log.Debugf("Instance status on delete is %s", status)
	if status != InstanceStatusStopped {
		log.Warningf("DeleteVM: vm status is %s expect %s", status, InstanceStatusStopped)
	}
	return self.doDeleteVM(instanceId)
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}

	// 修改密钥时直接返回
	if deleteKeypair {
		for i := 0; i < len(instance.LoginSettings.KeyIds); i++ {
			err = self.DetachKeyPair(instanceId, instance.LoginSettings.KeyIds[i])
			if err != nil {
				return err
			}
		}
	}

	if len(keypairName) > 0 {
		err = self.AttachKeypair(instanceId, keypairName)
		if err != nil {
			return err
		}
	}

	params := make(map[string]string)

	if len(name) > 0 && instance.InstanceName != name {
		params["InstanceName"] = name
	}

	if len(params) > 0 {
		err := self.modifyInstanceAttribute(instanceId, params)
		if err != nil {
			return err
		}
	}
	if len(password) > 0 {
		return self.instanceOperation(instanceId, "ResetInstancesPassword", map[string]string{"Password": password})
	}
	return nil
}

func (self *SInstance) DeleteVM() error {
	for {
		err := self.host.zone.region.DeleteVM(self.InstanceId)
		if err != nil {
			// if isError(err, "IncorrectInstanceStatus.Initializing") {
			// 	log.Infof("The instance is initializing, try later ...")
			// 	time.Sleep(10 * time.Second)
			// } else {
			// 	return err
			// }
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes
}

func (self *SRegion) UpdateVM(instanceId string, hostname string) error {
	params := make(map[string]string)
	params["HostName"] = hostname
	return self.modifyInstanceAttribute(instanceId, params)
}

func (self *SRegion) modifyInstanceAttribute(instanceId string, params map[string]string) error {
	return self.instanceOperation(instanceId, "ModifyInstancesAttribute", params)
}

func (self *SRegion) ReplaceSystemDisk(instanceId string, imageId string, passwd string, keypairName string, sysDiskSizeGB int) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["ImageId"] = imageId
	params["EnhancedService.SecurityService.Enabled"] = "TRUE"
	params["EnhancedService.MonitorService.Enabled"] = "TRUE"
	if len(passwd) > 0 {
		params["LoginSettings.Password"] = passwd
	} else {
		params["LoginSettings.KeepImageLogin"] = "TRUE"
	}
	if len(keypairName) > 0 {
		params["LoginSettings.KeyIds.0"] = keypairName
	}
	if sysDiskSizeGB > 0 {
		params["SystemDisk.DiskSize"] = fmt.Sprintf("%d", sysDiskSizeGB)
	}
	_, err := self.cvmRequest("ResetInstance", params)
	return err
}

func (self *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	// todo: support change disk config?
	params := make(map[string]string)
	instanceTypes, e := self.GetMatchInstanceTypes(ncpu, vmem, 0, zoneId)
	if e != nil {
		return e
	}

	for _, instancetype := range instanceTypes {
		params["InstanceType"] = instancetype.InstanceType
		err := self.instanceOperation(instanceId, "ResetInstancesType", params)
		if err != nil {
			log.Errorf("Failed for %s: %s", instancetype.InstanceType, err)
		}
		return nil
	}

	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["DiskId"] = diskId
	log.Infof("Detach instance %s disk %s", instanceId, diskId)
	_, err := self.cvmRequest("DetachDisk", params)
	if err != nil {
		log.Errorf("DetachDisk %s to %s fail %s", diskId, instanceId, err)
		return err
	}

	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["DiskIds.0"] = diskId
	_, err := self.cbsRequest("AttachDisks", params)
	if err != nil {
		log.Errorf("AttachDisks %s to %s fail %s", diskId, instanceId, err)
		return err
	}
	return nil
}

func (self *SInstance) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error {
	// if vpc, err := self.getVpc(); err != nil {
	// 	return err
	// } else if len(secgroupId) == 0 {
	// 	for index, secgrpId := range self.SecurityGroupIds.SecurityGroupId {
	// 		if err := vpc.revokeSecurityGroup(secgrpId, self.InstanceId, index == 0); err != nil {
	// 			return err
	// 		}
	// 	}
	// } else if secgrpId, err := vpc.SyncSecurityGroup(secgroupId, name, rules); err != nil {
	// 	return err
	// } else if err := vpc.assignSecurityGroup(secgrpId, self.InstanceId); err != nil {
	// 	return err
	// } else {
	// 	for _, secgroupId := range self.SecurityGroupIds.SecurityGroupId {
	// 		if secgroupId != secgrpId {
	// 			if err := vpc.revokeSecurityGroup(secgroupId, self.InstanceId, false); err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// 	self.SecurityGroupIds.SecurityGroupId = []string{secgrpId}
	// }
	return nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.PublicIpAddresses) > 0 {
		eip := SEipAddress{region: self.host.zone.region}
		eip.AddressIp = self.PublicIpAddresses[0]
		eip.InstanceId = self.InstanceId
		eip.AddressId = self.InstanceId
		eip.AddressName = self.PublicIpAddresses[0]
		eip.AddressType = "WanIP"
		eip.AddressStatus = EIP_STATUS_INUSE
		return &eip, nil
	}
	return nil, nil
}

func (self *SInstance) GetBillingType() string {
	switch self.InstanceChargeType {
	case PrePaidInstanceChargeType:
		return models.BILLING_TYPE_PREPAID
	case PostPaidInstanceChargeType:
		return models.BILLING_TYPE_POSTPAID
	default:
		return models.BILLING_TYPE_PREPAID
	}
}

func (self *SInstance) GetExpiredAt() time.Time {
	return self.ExpiredTime
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}