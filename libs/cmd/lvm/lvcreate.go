package lvm

import (
	"strconv"
)

func (lvm *LVM) LVCreate(name, vgName, extents string, stripeCount, stripeSizeKB, readahead int) error {
	args := []string{"lvcreate", "-n", name, "-l", extents}
	if stripeCount > 1 {
		args = append(args, "-i", strconv.Itoa(stripeCount), "-I", strconv.Itoa(stripeSizeKB))
	}
	if readahead > 0 {
		args = append(args, "-r", strconv.Itoa(readahead))
	} else {
		args = append(args, "-r", "auto")
	}
	args = append(args, vgName)
	cm, err := lvm.Cmd.Command("sudo", args...)
	if err != nil {
		return err
	}
	defer cm.Close()
	return cm.Run()
}
