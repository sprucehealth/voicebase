package lvm

import (
	"bytes"
	"strconv"
)

type VolumeGroup struct {
	Name   string
	Access string
	Status int
	// InternalVolumeGroupNumber int
	// 5  maximum number of logical volumes
	// 6  current number of logical volumes
	// 7  open count of all logical volumes in this volume group
	// 8  maximum logical volume size
	// 9  maximum number of physical volumes
	// 10 current number of physical volumes
	// 11 actual number of physical volumes
	// 12 size of volume group in kilobytes
	// 13 physical extent size
	// 14 total number of physical extents for this volume group
	// 15 allocated number of physical extents for this volume group
	// 16 free number of physical extents for this volume group
	// 17 uuid of volume group
}

func (lvm *LVM) VGDisplay() (map[string]*VolumeGroup, error) {
	buf := &bytes.Buffer{}
	cm, err := lvm.Cmd.Command("sudo", "vgdisplay", "-c")
	if err != nil {
		return nil, err
	}
	defer cm.Close()
	cm.Stdout = buf
	if err := cm.Run(); err != nil {
		return nil, err
	}
	return parseVGDisplay(buf.Bytes())
}

func parseVGDisplay(buf []byte) (map[string]*VolumeGroup, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	infos := make(map[string]*VolumeGroup)
	for _, line := range lines {
		parts := bytes.Split(bytes.TrimSpace(line), []byte(":"))
		if len(parts) < 6 {
			// Most likely "No volume groups found"
			continue
		}
		vg := &VolumeGroup{
			Name:   string(parts[0]),
			Access: string(parts[1]),
		}
		var err error
		if vg.Status, err = strconv.Atoi(string(parts[2])); err != nil {
			return nil, err
		}
		infos[vg.Name] = vg
	}
	return infos, nil
}
