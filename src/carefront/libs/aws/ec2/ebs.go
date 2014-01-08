package ec2

import (
	"fmt"
	"net/url"
	"strconv"
)

func (ec2 *EC2) AttachVolume(volumeId, instanceId, device string) (*AttachVolumeResponse, error) {
	params := url.Values{}
	params.Set("VolumeId", volumeId)
	params.Set("InstanceId", instanceId)
	params.Set("Device", device)
	res := &AttachVolumeResponse{}
	err := ec2.Get("AttachVolume", params, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ec2 *EC2) CreateSnapshot(volumeId, description string) (*CreateSnapshotResponse, error) {
	params := url.Values{}
	params.Set("VolumeId", volumeId)
	if description != "" {
		params.Set("Description", description)
	}
	res := &CreateSnapshotResponse{}
	err := ec2.Get("CreateSnapshot", params, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ec2 *EC2) CreateVolume(size int, az, volumeType, snapshotId string, iops int) (*CreateVolumeResponse, error) {
	params := url.Values{}
	if size > 0 { // When creating from a snapshot, the default is to use the snapshot size
		params.Set("Size", strconv.Itoa(size))
	}
	params.Set("AvailabilityZone", az)
	if snapshotId != "" {
		params.Set("SnapshotId", snapshotId)
	}
	if volumeType != "" {
		params.Set("VolumeType", volumeType)
	}
	if iops > 0 {
		params.Set("Iops", strconv.Itoa(iops))
		if volumeType == "" {
			params.Set("VolumeType", "io1")
		}
	}
	res := &CreateVolumeResponse{}
	err := ec2.Get("CreateVolume", params, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ec2 *EC2) DeleteSnapshot(id string) error {
	params := url.Values{
		"SnapshotId": []string{id},
	}
	res := &SimpleResponse{}
	err := ec2.Get("DeleteSnapshot", params, res)
	if err != nil {
		return err
	}
	if !res.Return {
		return fmt.Errorf("aws/ec2: operation failed")
	}
	return nil
}

func (ec2 *EC2) DescribeSnapshots(ids, owners, restorableBy []string, filters map[string][]string) ([]*Snapshot, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("SnapshotId.%d", i+1), id)
	}
	for i, id := range owners {
		params.Set(fmt.Sprintf("Owner.%d", i+1), id)
	}
	for i, id := range restorableBy {
		params.Set(fmt.Sprintf("RestorableBy.%d", i+1), id)
	}
	i := 1
	for name, values := range filters {
		params.Set(fmt.Sprintf("Filter.%d.Name", i), name)
		for j, val := range values {
			params.Set(fmt.Sprintf("Filter.%d.Value.%d", i, j+1), val)
		}
		i++
	}
	res := &DescribeSnapshotsResponse{}
	err := ec2.Get("DescribeSnapshots", params, res)
	if err != nil {
		return nil, err
	}
	return res.Snapshots, nil
}

func (ec2 *EC2) DescribeVolumes(ids []string, filters map[string][]string) ([]*Volume, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("VolumeId.%d", i+1), id)
	}
	i := 1
	for name, values := range filters {
		params.Set(fmt.Sprintf("Filter.%d.Name", i), name)
		for j, val := range values {
			params.Set(fmt.Sprintf("Filter.%d.Value.%d", i, j+1), val)
		}
		i++
	}
	res := &DescribeVolumesResponse{}
	err := ec2.Get("DescribeVolumes", params, res)
	if err != nil {
		return nil, err
	}
	return res.Volumes, nil
}

func (ec2 *EC2) DetachVolume(volumeId, instanceId, device string, force bool) (*AttachVolumeResponse, error) {
	params := url.Values{}
	params.Set("VolumeId", volumeId)
	params.Set("InstanceId", instanceId)
	params.Set("Device", device)
	if force {
		params.Set("Force", "true")
	}
	res := &AttachVolumeResponse{}
	err := ec2.Get("DetachVolume", params, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
