package huawei

// 华为云返回的时间格式
const DATETIME_FORMAT = "2006-01-02T15:04:05.999999999"

// Task status
const (
	TASK_SUCCESS = "SUCCESS"
	TASK_FAIL    = "FAIL"
)

// Charging Type
const (
	POST_PAID = "postPaid" // 按需付费
	PRE_PAID  = "prePaid"  // 包年包月
)

// 资源类型 https://support.huaweicloud.com/api-oce/zh-cn_topic_0079291752.html
const (
	RESOURCE_TYPE_VM        = "hws.resource.type.vm"          // ECS虚拟机
	RESOURCE_TYPE_VOLUME    = "hws.resource.type.volume"      // EVS卷
	RESOURCE_TYPE_BANDWIDTH = "hws.resource.type.bandwidth"   // VPC带宽
	RESOURCE_TYPE_IP        = "hws.resource.type.ip"          // VPC公网IP
	RESOURCE_TYPE_IMAGE     = "hws.resource.type.marketplace" // 市场镜像
)

// Not Found Error code
// 网络等资源直接通过http code 404即可判断资源不存在。另外有些资源可能不是返回404这里单独列出来
const (
	VM_NOT_FOUND            = "Ecs.0114"    // 云服务器不存在
	ECS_NOT_FOUND           = "Ecs.0614"    // 弹性云服务器不存在
	IMG_ID_NOT_FOUND        = "IMG.0027"    // 请求的镜像ID不存在
	IMG_NOT_FOUND           = "IMG.0027"    // 镜像不存在
	IMG_ERR_NOT_FOUND       = "IMG.0057"    // 镜像文件不存在或者为空或者不是允许格式的文件
	IMG_BACKUP_NOT_FOUND    = "IMG.0020"    // 备份不存在
	IMG_VM_BACKUP_NOT_FOUND = "IMG.0127"    // 云服务器备份不存在
	IMG_VM_NOT_FOUND        = "IMG.0005"    // 云主机不存在
	JOB_NOT_FOUND           = "Common.0011" // jobId为空
	EVS_NOT_FOUND           = "EVS.5404"    // 磁盘、快照和备份等资源未找到。
	FIP_NOT_FOUND           = "VPC.0504"    // 未找到弹性公网IP。
	VPC_NOT_FOUND           = "VPC.0012"    // 未找到弹性公网VPC。
)

var NOT_FOUND_CODES = []string{
	VM_NOT_FOUND,
	ECS_NOT_FOUND,
	IMG_ID_NOT_FOUND,
	IMG_NOT_FOUND,
	IMG_ERR_NOT_FOUND,
	IMG_BACKUP_NOT_FOUND,
	IMG_VM_BACKUP_NOT_FOUND,
	IMG_VM_NOT_FOUND,
	JOB_NOT_FOUND,
	EVS_NOT_FOUND,
	FIP_NOT_FOUND,
	VPC_NOT_FOUND,
}