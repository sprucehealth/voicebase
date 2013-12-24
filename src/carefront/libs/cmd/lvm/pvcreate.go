package lvm

func (lvm *LVM) PVCreate(device string) error {
	cm, err := lvm.Cmd.Command("sudo", "pvcreate", device)
	if err != nil {
		return err
	}
	defer cm.Close()
	return cm.Run()
}
