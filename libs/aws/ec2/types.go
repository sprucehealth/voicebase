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
	ID   string `xml:"groupId"`
	Name string `xml:"groupName"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-UserIdGroupPairType.html
type UserGroup struct {
	UserID string `xml:"userId"`
	ID     string `xml:"groupId"`
	Name   string `xml:"groupName"`
}

type VolumeAttachment struct {
	VolumeID            string `xml:"volumeId"`
	InstanceID          string `xml:"instanceId"`
	Device              string `xml:"device"`
	Status              string `xml:"status"` // attaching | attached | detaching | detached
	AttachTime          Time   `xml:"attachTime"`
	DeleteOnTermination bool   `xml:"deleteOnTermination"`
}

// type NetworkInterface struct {
// 	Status      string `xml:"status"`
// 	OwnerID     string `xml:"ownerId"`
// 	Description string `xml:"description"`
// 	VpcID              string `xml:"vpcId"`
// 	SubnetID           string `xml:"subnetId"`
// 	NetworkInterfaceID string `xml:"networkInterfaceId"`
// }

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-ProductCodesSetItemType.html
type ProductCode struct {
	ProductCode string `xml:"productCode"`
	Type        string `xml:"type"` // devpay | marketplace
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-EbsInstanceBlockDeviceMappingResponseType.html
type EBS struct {
	VolumeID            string `xml:"volumeId"`
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
	InstanceID       string        `xml:"instanceId"`
	ImageID          string        `xml:"imageId"`
	InstanceState    string        `xml:"instanceState"`
	PrivateDNSName   string        `xml:"privateDnsName"`
	DNSName          string        `xml:"dnsName"`
	Reason           string        `xml:"reason"`
	KeyName          string        `xml:"keyName"`
	AmiLaunchIndex   int           `xml:"amiLaunchIndex"`
	ProductCodes     []ProductCode `xml:"productCodes>item"`
	InstanceType     string        `xml:"instanceType"`
	LaunchTime       Time          `xml:"launchTime"`
	Placement        string        `xml:"placement"`
	KernelID         string        `xml:"kernelId"`
	RamdiskID        string        `xml:"ramdiskId"`
	Platform         string        `xml:"platform"`
	Monitoring       string        `xml:"monitoring"`
	SubnetID         string        `xml:"subnetId"`
	VpcID            string        `xml:"vpcId"`
	PrivateIPAddress string        `xml:"privateIpAddress"`
	IPAddress        string        `xml:"ipAddress"`
	SourceDestCheck  bool          `xml:"sourceDestCheck"`
	Groups           []Group       `xml:"groupSet>item"`
	// stateReason
	Architecture          string         `xml:"architecture"`   // i386 | x86_64
	RootDeviceType        string         `xml:"rootDeviceType"` // ebs | instance-store
	RootDeviceName        string         `xml:"rootDeviceName"`
	BlockDevices          []*BlockDevice `xml:"blockDeviceMapping"`
	InstanceLifecycle     string         `xml:"instanceLifecycle"` // spot | blank (no value)
	SpotInstanceRequestID string         `xml:"spotInstanceRequestId"`
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
	ReservationID string      `xml:"reservationId"`
	OwnerID       string      `xml:"ownerId"`
	Groups        []Group     `xml:"groupSet>item"`
	Instances     []*Instance `xml:"instancesSet>item"`
	RequesterID   string      `xml:"requesterId"`
}

type Snapshot struct {
	SnapshotID  string `xml:"snapshotId"`
	VolumeID    string `xml:"volumeId"`
	Status      string `xml:"status"` // pending, completed, error
	StartTime   Time   `xml:"startTime"`
	Progress    string `xml:"progress"` // percentage
	OwnerID     string `xml:"ownerId"`
	VolumeSize  int    `xml:"volumeSize"` // GiB
	Description string `xml:"description"`
	OwnerAlias  string `xml:"ownerAlias"`
	Tags        Tags   `xml:"tagSet"`
}

type Volume struct {
	VolumeID         string            `xml:"volumeId"`
	Size             int               `xml:"size"` // GiB
	SnapshotID       string            `xml:"snapshotId"`
	AvailabilityZone string            `xml:"availabilityZone"`
	Status           string            `xml:"status"`
	CreateTime       Time              `xml:"createTime"`
	VolumeType       string            `xml:"volumeType"`
	Iops             int               `xml:"iops"`
	Attachment       *VolumeAttachment `xml:"attachmentSet>item"`
	Tags             Tags              `xml:"tagSet"`
}

type AttachVolumeResponse struct {
	RequestID  string `xml:"requestId"`
	VolumeID   string `xml:"volumeId"`
	InstanceID string `xml:"instanceId"`
	Device     string `xml:"device"`
	Status     string `xml:"status"` // attaching | attached | detaching | detached
	AttachTime Time   `xml:"attachTime"`
}

type CreateSnapshotResponse struct {
	RequestID   string `xml:"requestId"`
	SnapshotID  string `xml:"snapshotId"`
	VolumeID    string `xml:"volumeId"`
	Status      string `xml:"status"` // pending, completed, error
	StartTime   Time   `xml:"startTime"`
	Progress    string `xml:"progress"` // percentage
	OwnerID     string `xml:"ownerId"`
	VolumeSize  int    `xml:"volumeSize"` // GiB
	Description string `xml:"description"`
}

type CreateVolumeResponse struct {
	RequestID        string `xml:"requestId"`
	VolumeID         string `xml:"volumeId"`
	Size             int    `xml:"size"` // GiB
	SnapshotID       string `xml:"snapshotId"`
	AvailabilityZone string `xml:"availabilityZone"`
	Status           string `xml:"status"`
	CreateTime       Time   `xml:"createTime"`
	VolumeType       string `xml:"volumeType"`
	Iops             int    `xml:"iops"`
}

type CreateTagsResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type DescribeInstancesResponse struct {
	RequestID    string         `xml:"requestId"`
	Reservations []*Reservation `xml:"reservationSet>item"`
	NextToken    string         `xml:"nextToken"`
}

type NetworkACLEntry struct {
	RoleNumber int    `xml:"roleNumber"`
	Protocol   string `xml:"protocol"`   // all, ...
	RuleAction string `xml:"ruleAction"` // allow, deny
	Egress     bool   `xml:"egress"`
	CIDRBlock  string `xml:"cidrBlock"`
}

type NetworkACLAssociation struct {
	NetworkACLAssociationID string `xml:"networkAclAssociationId"`
	NetworkACLID            string `xml:"networkAclId"`
	SubnetID                string `xml:"subnetId"`
}

type NetworkACL struct {
	NetworkACLID string                   `xml:"networkAclId"`
	VPCID        string                   `xml:"vpcId"`
	Default      bool                     `xml:"default"`
	Entries      []*NetworkACLEntry       `xml:"entrySet>item"`
	Associations []*NetworkACLAssociation `xml:"associationSet>item"`
	Tags         Tags                     `xml:"tagSet"`
}

type DescribeNetworkACLsResponse struct {
	RequestID   string        `xml:"requestId"`
	NetworkACLs []*NetworkACL `xml:"networkAclSet>item"`
}

type Route struct {
	DestinationCIDRBlock string `xml:"destinationCidrBlock"`
	GatewayID            string `xml:"gatewayId"`
	State                string `xml:"state"`
	Origin               string `xml:"origin"`
}

type RouteAssociation struct {
	RouteTableAssociationID string `xml:"routeTableAssociationId"`
	RouteTableID            string `xml:"routeTableId"`
	Main                    bool   `xml:"main"`
	SubnetID                string `xml:"subnetId"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-RouteTableType.html
type RouteTable struct {
	RouteTableID    string              `xml:"routeTableId"`
	VPCID           string              `xml:"vpcId"`
	Routes          []*Route            `xml:"routeSet>item"`
	Associations    []*RouteAssociation `xml:"associationSet>item"`
	PropagatingVGWs []string            `xml:"propagatingVgwSet>item>gatewayID"`
	Tags            Tags                `xml:"tagSet"`
}

type DescribeRouteTablesResponse struct {
	RequestID   string        `xml:"requestId"`
	RouteTables []*RouteTable `xml:"routeTableSet>item"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-IpPermissionType.html
type IPPermission struct {
	IPProtocol string       `xml:"ipProtocol"`
	FromPort   int          `xml:"fromPort"`
	ToPort     int          `xml:"toPort"`
	Groups     []*UserGroup `xml:"groups>item"`
	IPRanges   []string     `xml:"ipRanges>item>cidrIp"` // http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-IpRangeItemType.html
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-SecurityGroupItemType.html
type SecurityGroup struct {
	OwnerID             string          `xml:"ownerId"`
	GroupID             string          `xml:"groupId"`
	GroupName           string          `xml:"groupName"`
	GroupDescription    string          `xml:"groupDescription"`
	VPCID               string          `xml:"vpcId"`
	IPPermissions       []*IPPermission `xml:"ipPermissions>item"`
	IPPermissionsEgress []*IPPermission `xml:"ipPermissionsEgress>item"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-DescribeSecurityGroups.html
type DescribeSecurityGroupsResponse struct {
	RequestID      string           `xml:"requestId"`
	SecurityGroups []*SecurityGroup `xml:"securityGroupInfo>item"`
}

type DescribeSnapshotsResponse struct {
	RequestID string      `xml:"requestId"`
	Snapshots []*Snapshot `xml:"snapshotSet>item"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-SubnetType.html
type Subnet struct {
	SubnetID                string `xml:"subnetId"`
	State                   string `xml:"state"` // pending, available
	VPDID                   string `xml:"vpcId"`
	CIDRBlock               string `xml:"cidrBlock"`
	AvailableIPAddressCount int    `xml:"availableIpAddressCount"`
	AvailabilityZone        string `xml:"availabilityZone"`
	DefaultForAZ            bool   `xml:"defaultForAz"`
	MapPublicIPOnLaunch     bool   `xml:"mapPublicIpOnLaunch"`
	Tags                    Tags   `xml:"tagSet"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-DescribeSubnets.html
type DescribeSubnetsResponse struct {
	RequestID string    `xml:"requestId"`
	Subnets   []*Subnet `xml:"subnetSet>item"`
}

type DescribeVolumesResponse struct {
	RequestID string    `xml:"requestId"`
	Volumes   []*Volume `xml:"volumeSet>item"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-VpcType.html
type VPC struct {
	VPCID           string `xml:"vpcId"`
	State           string `xml:"state"` // pending | available
	CIDRBlock       string `xml:"cidrBlock"`
	DHCPOptionsID   string `xml:"dhcpOptionsId"`
	Tags            Tags   `xml:"tagSet"`
	InstanceTenancy string `xml:"instanceTenancy"` // default | dedicated
	IsDefault       bool   `xml:"isDefault"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-DescribeVpcs.html
type DescribeVPCsResponse struct {
	RequestID string `xml:"requestId"`
	VPCs      []*VPC `xml:"vpcSet>item"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-VpcPeeringConnectionVpcInfoType.html
type VPCPeeringConnectionVPCInfo struct {
	VPCID     string `xml:"vpcId"`
	OwnerID   string `xml:"ownerId"`
	CIDRBlock string `xml:"cidrBlock"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-VpcPeeringConnectionStateReasonType.html
type VPCPeeringConnectionStateReason struct {
	Code    string `xml:"code"` // initiating-request | pending-acceptance | failed | expired | provisioning | active | deleted | rejected
	Message string `xml:"message"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-VpcPeeringConnectionType.html
type VPCPeeringConnection struct {
	VPCPeeringConnectionID string                           `xml:"vpcPeeringConnectionId"`
	RequesterVPCInfo       *VPCPeeringConnectionVPCInfo     `xml:"requesterVpcInfo"`
	AccepterVPCInfo        *VPCPeeringConnectionVPCInfo     `xml:"accepterVpcInfo"`
	Status                 *VPCPeeringConnectionStateReason `xml:"status"`
	ExpirationTime         Time                             `xml:"expirationTime"`
	Tags                   Tags                             `xml:"tagSet"`
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-DescribeVpcPeeringConnections.html
type DescribeVPCPeeringConnectionsResponse struct {
	RequestID   string                  `xml:"requestId"`
	Connections []*VPCPeeringConnection `xml:"vpcPeeringConnectionSet>item"`
}

type SimpleResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}
