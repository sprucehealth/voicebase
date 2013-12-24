package lvm

import (
	"bytes"
	"fmt"

	"carefront/libs/cmd"
)

type LVInfo struct {
	VGName  string
	Name    string
	Type    string
	Devices []string
}

type LVM struct {
	Cmd cmd.Commander
}

var Default = &LVM{Cmd: cmd.LocalCommander}

func (lvm *LVM) LVS(path string) (*LVInfo, error) {
	buf := &bytes.Buffer{}
	cm, err := lvm.Cmd.Command("sudo", "lvs", "--segments", "-o", "+devices", "--all", "--separator", "|", path)
	if err != nil {
		return nil, err
	}
	defer cm.Close()
	cm.Stdout = buf
	if err := cm.Run(); err != nil {
		return nil, err
	}
	info, err := parseLVS(buf.Bytes())
	if err != nil {
		return nil, err
	}
	if len(info) > 0 {
		return info[0], nil
	}
	return nil, fmt.Errorf("Failed to get lvs info")
}

func parseLVS(buf []byte) ([]*LVInfo, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	infos := make([]*LVInfo, 0, len(lines)-1)
	var colIndex []string
	for i, line := range lines {
		parts := bytes.Split(line, []byte("|"))
		if i == 0 {
			// First line is the header
			colIndex = make([]string, len(parts))
			for j, c := range parts {
				colIndex[j] = string(c)
			}
			continue
		}
		info := &LVInfo{}
		for j, p := range parts {
			var err error
			switch colIndex[j] {
			case "LV":
				info.Name = string(p)
			case "VG":
				info.VGName = string(p)
			case "Type":
				info.Type = string(p)
			case "Devices":
				devs := bytes.Split(p, []byte(","))
				info.Devices = make([]string, len(devs))
				for i, d := range devs {
					idx := bytes.IndexByte(d, '(')
					if idx < 0 {
						idx = len(d)
					}
					info.Devices[i] = string(d[:idx])
				}
			}
			if err != nil {
				return nil, err
			}
		}
		if info.Name != "" {
			infos = append(infos, info)
		}
	}
	return infos, nil
}
