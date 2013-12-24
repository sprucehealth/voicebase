package ec2

import (
	"encoding/xml"
	"time"
)

type Tags map[string]string

type keyValue struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

func (t *Tags) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var kv struct {
		KV []keyValue `xml:"item"`
	}
	if err := d.DecodeElement(&kv, &start); err != nil {
		return err
	}
	if *t == nil {
		*t = Tags(make(map[string]string))
	}
	for _, kv := range kv.KV {
		(*t)[kv.Key] = kv.Value
	}
	return nil
}

type Time time.Time

func (t Time) String() string {
	return time.Time(t).String()
}

func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var timeStr string
	if err := d.DecodeElement(&timeStr, &start); err != nil {
		return err
	}
	tm, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return err
	}
	*t = Time(tm)
	return nil
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-GroupItemType.html
type Group struct {
	Id   string `xml:"groupId"`
	Name string `xml:"groupName"`
}

type VolumeAttachment struct {
	VolumeId            string `xml:"volumeId"`
	InstanceId          string `xml:"instanceId"`
	Device              string `xml:"device"`
	Status              string `xml:"status"` // attaching | attached | detaching | detached
	AttachTime          Time   `xml:"attachTime"`
	DeleteOnTermination bool   `xml:"deleteOnTermination"`
}

// type NetworkInterface struct {
// 	Status      string `xml:"status"`
// 	OwnerId     string `xml:"ownerId"`
// 	Description string `xml:"description"`
// 	VpcId              string `xml:"vpcId"`
// 	SubnetId           string `xml:"subnetId"`
// 	NetworkInterfaceId string `xml:"networkInterfaceId"`
// }

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-ProductCodesSetItemType.html
type ProductCode struct {
	ProductCode string `xml:"productCode"`
	Type        string `xml:"type"` // devpay | marketplace
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-EbsInstanceBlockDeviceMappingResponseType.html
type EBS struct {
	VolumeId            string `xml:"volumeId"`
	Status              string `xml:"status"` // attaching | attached | detaching | detached
	AttachTime          Time   `xml:"attachTime"`
	DeleteOnTermination bool   `xml:"deleteOnTermination"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-InstanceBlockDeviceMappingResponseItemType.html
type BlockDevice struct {
	DeviceName string `xml:"deviceName"`
	EBS        EBS    `xml:"ebs"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-RunningInstancesItemType.html
type Instance struct {
	InstanceId       string        `xml:"instanceId"`
	ImageId          string        `xml:"imageId"`
	InstanceState    string        `xml:"instanceState"`
	PrivateDnsName   string        `xml:"privateDnsName"`
	DnsName          string        `xml:"dnsName"`
	Reason           string        `xml:"reason"`
	KeyName          string        `xml:"keyName"`
	AmiLaunchIndex   int           `xml:"amiLaunchIndex"`
	ProductCodes     []ProductCode `xml:"productCodes>item"`
	InstanceType     string        `xml:"instanceType"`
	LaunchTime       Time          `xml:"launchTime"`
	Placement        string        `xml:"placement"`
	KernelId         string        `xml:"kernelId"`
	RamdiskId        string        `xml:"ramdiskId"`
	Platform         string        `xml:"platform"`
	Monitoring       string        `xml:"monitoring"`
	SubnetId         string        `xml:"subnetId"`
	VpcId            string        `xml:"vpcId"`
	PrivateIpAddress string        `xml:"privateIpAddress"`
	IpAddress        string        `xml:"ipAddress"`
	SourceDestCheck  bool          `xml:"sourceDestCheck"`
	Groups           []Group       `xml:"groupSet>item"`
	// stateReason
	Architecture          string         `xml:"architecture"`   // i386 | x86_64
	RootDeviceType        string         `xml:"rootDeviceType"` // ebs | instance-store
	RootDeviceName        string         `xml:"rootDeviceName"`
	BlockDevices          []*BlockDevice `xml:"blockDeviceMapping"`
	InstanceLifecycle     string         `xml:"instanceLifecycle"` // spot | blank (no value)
	SpotInstanceRequestId string         `xml:"spotInstanceRequestId"`
	VirtualizationType    string         `xml:"virtualizationType"` // paravirtual | hvm
	ClientToken           string         `xml:"clientToken"`
	Tags                  Tags           `xml:"tagSet"`
	Hypervisor            string         `xml:"hypervisor"` // ovm | xen
	// networkInterfaceSet
	// iamInstanceProfile
	EbsOptimized    bool   `xml:"ebsOptimized"`
	SriovNetSupport string `xml:"sriovNetSupport"` // simple
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-ReservationInfoType.html
type Reservation struct {
	ReservationId string      `xml:"reservationId"`
	OwnerId       string      `xml:"ownerId"`
	Groups        []Group     `xml:"groupSet>item"`
	Instances     []*Instance `xml:"instancesSet>item"`
	RequesterId   string      `xml:"requesterId"`
}

type Snapshot struct {
	SnapshotId  string `xml:"snapshotId"`
	VolumeId    string `xml:"volumeId"`
	Status      string `xml:"status"` // pending, completed, error
	StartTime   Time   `xml:"startTime"`
	Progress    string `xml:"progress"` // percentage
	OwnerId     string `xml:"ownerId"`
	VolumeSize  int    `xml:"volumeSize"` // GiB
	Description string `xml:"description"`
	OwnerAlias  string `xml:"ownerAlias"`
	Tags        Tags   `xml:"tagSet"`
}

type Volume struct {
	VolumeId         string            `xml:"volumeId"`
	Size             int               `xml:"size"` // GiB
	SnapshotId       string            `xml:"snapshotId"`
	AvailabilityZone string            `xml:"availabilityZone"`
	Status           string            `xml:"status"`
	CreateTime       Time              `xml:"createTime"`
	VolumeType       string            `xml:"volumeType"`
	Iops             int               `xml:"iops"`
	Attachment       *VolumeAttachment `xml:"attachmentSet>item"`
	Tags             Tags              `xml:"tagSet"`
}

type AttachVolumeResponse struct {
	RequestId  string `xml:"requestId"`
	VolumeId   string `xml:"volumeId"`
	InstanceId string `xml:"instanceId"`
	Device     string `xml:"device"`
	Status     string `xml:"status"` // attaching | attached | detaching | detached
	AttachTime Time   `xml:"attachTime"`
}

type CreateSnapshotResponse struct {
	RequestId   string `xml:"requestId"`
	SnapshotId  string `xml:"snapshotId"`
	VolumeId    string `xml:"volumeId"`
	Status      string `xml:"status"` // pending, completed, error
	StartTime   Time   `xml:"startTime"`
	Progress    string `xml:"progress"` // percentage
	OwnerId     string `xml:"ownerId"`
	VolumeSize  int    `xml:"volumeSize"` // GiB
	Description string `xml:"description"`
}

type CreateVolumeResponse struct {
	RequestId        string `xml:"requestId"`
	VolumeId         string `xml:"volumeId"`
	Size             int    `xml:"size"` // GiB
	SnapshotId       string `xml:"snapshotId"`
	AvailabilityZone string `xml:"availabilityZone"`
	Status           string `xml:"status"`
	CreateTime       Time   `xml:"createTime"`
	VolumeType       string `xml:"volumeType"`
	Iops             int    `xml:"iops"`
}

type CreateTagsResponse struct {
	RequestId string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type DescribeInstancesResponse struct {
	RequestId    string         `xml:"requestId"`
	Reservations []*Reservation `xml:"reservationSet>item"`
	NextToken    string         `xml:"nextToken"`
}

type DescribeSnapshotsResponse struct {
	RequestId string      `xml:"requestId"`
	Snapshots []*Snapshot `xml:"snapshotSet>item"`
}

type DescribeVolumesResponse struct {
	RequestId string    `xml:"requestId"`
	Volumes   []*Volume `xml:"volumeSet>item"`
}
