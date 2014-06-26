package mount

import (
	"bytes"
)

func (m *MountCmd) GetMounts() (map[string]*Mount, error) {
	buf := &bytes.Buffer{}
	cmd, err := m.Cmd.Command("mount", "-l")
	if err != nil {
		return nil, err
	}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return parseLinuxMounts(buf.Bytes())
}
