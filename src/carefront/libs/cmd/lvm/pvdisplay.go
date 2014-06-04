package lvm

import (
	"bytes"
	"os"
)

type PhysicalVolume struct {
	Name   string
	VGName string
	// * physical volume size in kilobytes
	// * internal physical volume number (obsolete)
	// * physical volume status
	// * physical volume (not) allocatable
	// * current number of logical volumes on this physical volume
	// * physical extent size in kilobytes
	// * total number of physical extents
	// * free number of physical extents
	// * allocated number of physical extents
}

func (lvm *LVM) PVDisplay() (map[string]*PhysicalVolume, error) {
	buf := &bytes.Buffer{}
	cm, err := lvm.Cmd.Command("sudo", "pvdisplay", "-c")
	if err != nil {
		return nil, err
	}
	defer cm.Close()
	cm.Stdout = buf
	cm.Stderr = os.Stderr
	if err := cm.Run(); err != nil {
		return nil, err
	}
	return parsePVDisplay(buf.Bytes())
}

func parsePVDisplay(buf []byte) (map[string]*PhysicalVolume, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	infos := make(map[string]*PhysicalVolume)
	for _, line := range lines {
		parts := bytes.Split(bytes.TrimSpace(line), []byte(":"))
		if len(parts) < 3 {
			continue
		}
		pv := &PhysicalVolume{
			Name:   string(parts[0]),
			VGName: string(parts[1]),
		}
		infos[pv.Name] = pv
	}
	return infos, nil
}
