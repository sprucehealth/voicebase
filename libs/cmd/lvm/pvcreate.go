package lvm

import "os"

func (lvm *LVM) PVCreate(device string) error {
	cm, err := lvm.Cmd.Command("sudo", "pvcreate", device)
	if err != nil {
		return err
	}
	defer cm.Close()
	cm.Stderr = os.Stderr
	return cm.Run()
}
