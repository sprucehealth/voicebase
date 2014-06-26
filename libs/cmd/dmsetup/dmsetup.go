package dmsetup

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/libs/cmd"
)

type DeviceInfo struct {
	Name    string
	Major   int    // Major device number
	Minor   int    // Minor device number
	Stat    string // TODO: should be parsed: 'L--w'
	Open    int    // Open reference count
	Targets int    // Number of targets in the live table
	Event   int    // Last event sequence number (used by wait)
	UUID    string
}

type DMSetup struct {
	Cmd cmd.Commander
}

var Default = &DMSetup{Cmd: cmd.LocalCommander}

func (dm *DMSetup) DMInfo(device string) (*DeviceInfo, error) {
	buf := &bytes.Buffer{}
	c, err := dm.Cmd.Command("dmsetup", "-c", "info", "--separator", "|", device)
	if err != nil {
		return nil, err
	}
	c.Stdout = buf
	if err := c.Run(); err != nil {
		return nil, err
	}
	devs, err := parseDMInfo(buf.Bytes())
	if err != nil {
		return nil, err
	}
	for _, dev := range devs {
		return dev, nil
	}
	return nil, fmt.Errorf("Failed to get device")
}

func parseDMInfo(buf []byte) (map[string]*DeviceInfo, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	devices := make(map[string]*DeviceInfo, len(lines))
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
		dev := &DeviceInfo{}
		for j, p := range parts {
			var err error
			switch colIndex[j] {
			case "Name":
				dev.Name = string(p)
			case "Maj":
				dev.Major, err = strconv.Atoi(string(p))
			case "Min":
				dev.Minor, err = strconv.Atoi(string(p))
			case "Stat":
				dev.Stat = string(p)
			case "Open":
				dev.Open, err = strconv.Atoi(string(p))
			case "Targ":
				dev.Targets, err = strconv.Atoi(string(p))
			case "Event":
				dev.Event, err = strconv.Atoi(string(p))
			case "UUID":
				dev.UUID = string(p)
			}
			if err != nil {
				return nil, err
			}
			devices[dev.Name] = dev
		}
	}
	return devices, nil
}
