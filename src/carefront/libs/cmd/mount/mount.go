package mount

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"carefront/libs/cmd"
)

var (
	linuxMountRE  = regexp.MustCompile(`^(.+) on (.+) type (.+) \((.+)\)( \[(.+)\])?`)
	darwinMountRE = regexp.MustCompile(`^(.+) on (.+) \((.+)\)`)
)

type Mount struct {
	Device  string
	Path    string
	Type    string
	Label   string
	Options map[string]string
}

type MountCmd struct {
	Cmd cmd.Commander
}

var (
	ErrAlreadyMounted = errors.New("mount: already mounted")
)

type ErrMountPointDoesNotExist string

func (e ErrMountPointDoesNotExist) Error() string { return string(e) }

type ErrDeviceDoesNotExist string

func (e ErrDeviceDoesNotExist) Error() string { return string(e) }

var Default = &MountCmd{Cmd: cmd.LocalCommander}

func (m *MountCmd) Mount(device, path string) error {
	c, err := m.Cmd.Command("sudo", "mount", device, path)
	if err != nil {
		return err
	}
	defer c.Close()
	buf := &bytes.Buffer{}
	c.Stderr = buf
	if err := c.Run(); err == nil {
		return nil
	} else if e, ok := err.(*cmd.ExitError); ok && e.Status == 32 {
		by := buf.Bytes()
		if bytes.Contains(by, []byte("does not exist")) {
			if bytes.Contains(by, []byte("mount point")) {
				return ErrMountPointDoesNotExist(string(by))
			}
			if bytes.Contains(by, []byte("special device")) {
				return ErrDeviceDoesNotExist(string(by))
			}
		} else if bytes.Contains(by, []byte("already mounted")) {
			return ErrAlreadyMounted
		}
		return errors.New(string(by))
	} else {
		return err
	}
}

func parseLinuxMounts(buf []byte) (map[string]*Mount, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	mounts := make(map[string]*Mount, len(lines))
	for _, line := range lines {
		m := linuxMountRE.FindSubmatch(line)
		if len(m) > 0 {
			mount := &Mount{
				Device:  string(m[1]),
				Path:    string(m[2]),
				Type:    string(m[3]),
				Label:   string(m[6]),
				Options: make(map[string]string),
			}
			for _, opt := range bytes.Split(m[4], []byte(",")) {
				if idx := bytes.IndexByte(opt, '='); idx >= 0 {
					mount.Options[string(opt[:idx])] = string(opt[idx+1:])
				} else {
					mount.Options[string(opt)] = ""
				}
			}
			mounts[mount.Path] = mount
		} else {
			return mounts, fmt.Errorf("Failed to parse mount line: %s", string(line))
		}
	}
	return mounts, nil
}

func parseDarwinMounts(buf []byte) (map[string]*Mount, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	mounts := make(map[string]*Mount, len(lines))
	for _, line := range lines {
		m := darwinMountRE.FindSubmatch(line)
		if len(m) > 0 {
			mount := &Mount{
				Device:  string(m[1]),
				Path:    string(m[2]),
				Options: make(map[string]string),
			}
			for i, opt := range bytes.Split(m[3], []byte(", ")) {
				if i == 0 {
					// First option is the filesystem type
					mount.Type = string(opt)
					continue
				}
				if idx := bytes.IndexByte(opt, '='); idx >= 0 {
					mount.Options[string(opt[:idx])] = string(opt[idx+1:])
				} else {
					mount.Options[string(opt)] = ""
				}
			}
			mounts[mount.Path] = mount
		} else {
			return mounts, fmt.Errorf("Failed to parse mount line: %s", string(line))
		}
	}
	return mounts, nil
}
