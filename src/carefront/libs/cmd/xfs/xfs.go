package xfs

import (
	"bytes"

	"carefront/libs/cmd"
)

type XFS struct {
	Cmd cmd.Commander
}

func (xfs *XFS) IsXFS(device string) (isxfs bool, label string, uuid string, err error) {
	buf := &bytes.Buffer{}
	cm, er := xfs.Cmd.Command("sudo", "xfs_admin", "-lu", device)
	if er != nil {
		err = er
		return
	}
	defer cm.Close()
	cm.Stdout = buf
	cm.Stderr = buf
	if err = cm.Run(); err == nil {
		by := buf.Bytes()
		_ = by
		// TODO: parse label and UUID
		// label = "mysql-data"
		// UUID = 72880372-3f94-445b-aba1-7b0a3115d8e2
		isxfs = true
	} else if e, ok := err.(*cmd.ExitError); ok && e.Status == 1 {
		if bytes.Contains(buf.Bytes(), []byte("is not a valid XFS filesystem")) {
			err = nil
			isxfs = false
		}
	}
	return
}

func (xfs *XFS) Format(device string) error {
	cm, err := xfs.Cmd.Command("sudo", "mkfs.xfs", device)
	if err != nil {
		return err
	}
	defer cm.Close()
	return cm.Run()
}

func (xfs *XFS) SetLabel(device, label string) error {
	cm, err := xfs.Cmd.Command("sudo", "xfs_admin", "-L", label, device)
	if err != nil {
		return err
	}
	defer cm.Close()
	return cm.Run()
}
