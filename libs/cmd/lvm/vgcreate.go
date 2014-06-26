package lvm

func (lvm *LVM) VGCreate(name string, devices []string) error {
	args := append([]string{"vgcreate", name}, devices...)
	cm, err := lvm.Cmd.Command("sudo", args...)
	if err != nil {
		return err
	}
	defer cm.Close()
	return cm.Run()
}
