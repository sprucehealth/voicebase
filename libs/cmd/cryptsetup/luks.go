package cryptsetup

import (
	"bytes"
	"os"
	"strconv"

	"github.com/sprucehealth/backend/libs/cmd"
)

var DefaultCipher = "aes-xts-plain64"

type Status struct {
	Active  bool
	InUse   bool
	Type    string
	Cipher  string
	Keysize int // bits
	Device  string
	Offset  int64 // sectors
	Size    int64 // sectors
	Mode    string
}

type Cryptsetup struct {
	Cmd cmd.Commander
}

var Default = &Cryptsetup{Cmd: cmd.LocalCommander}

func (cs *Cryptsetup) cmd(name string, args ...string) (*cmd.Cmd, error) {
	c, err := cs.Cmd.Command("sudo", append([]string{"cryptsetup", name}, args...)...)
	if err != nil {
		return nil, err
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout
	return c, nil
}

func (cs *Cryptsetup) IsLuks(device string) (bool, error) {
	c, err := cs.cmd("isLuks", device)
	if err != nil {
		return false, err
	}
	defer c.Close()
	if err := c.Run(); err == nil {
		return true, nil
	} else if _, ok := err.(*cmd.ExitError); ok {
		// 1 = Device /dev/xxx is not a valid LUKS device.
		// 4 = Device /dev/xxx doesn't exist or access denied.
		return false, nil
	} else {
		return false, err
	}
}

func (cs *Cryptsetup) LuksOpen(name, device string, key []byte) error {
	c, err := cs.cmd("luksOpen", "--key-file", "-", device, name)
	if err != nil {
		return err
	}
	defer c.Close()
	c.Stdin = bytes.NewReader(key)
	if err := c.Run(); err == nil {
		return nil
	} else if e, ok := err.(*cmd.ExitError); ok && e.Status == 5 {
		return nil
	} else {
		return err
	}
}

func (cs *Cryptsetup) LuksClose(name string) error {
	c, err := cs.cmd("luksClose", name)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Run(); err == nil {
		return nil
	} else if e, ok := err.(*cmd.ExitError); ok && e.Status == 4 { // 4 = not open
		return nil
	} else {
		return err
	}
}

func (cs *Cryptsetup) LuksFormat(device, cipher string, key []byte) error {
	if cipher == "" {
		cipher = DefaultCipher
	}
	c, err := cs.cmd("luksFormat", "--cipher", cipher, device, "-")
	if err != nil {
		return err
	}
	defer c.Close()
	c.Stdin = bytes.NewReader(key)
	return c.Run()
}

func (cs *Cryptsetup) Status(device string) (*Status, error) {
	buf := &bytes.Buffer{}
	c, err := cs.cmd("status", device)
	if err != nil {
		return nil, err
	}
	c.Stdout = buf
	if err := c.Run(); err != nil {
		return nil, err
	}
	return parseStatus(buf.Bytes())
}

func parseStatus(buf []byte) (*Status, error) {
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	status := &Status{}
	for i, line := range lines {
		if i == 0 {
			// First line is a header
			status.Active = bytes.Contains(line, []byte("is active"))
			status.InUse = bytes.Contains(line, []byte("is in use"))
			continue
		}
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := string(bytes.TrimSpace(line[:idx]))
		value := bytes.TrimSpace(line[idx+1:])
		var err error
		switch key {
		case "type":
			status.Type = string(value)
		case "cipher":
			status.Cipher = string(value)
		case "device":
			status.Device = string(value)
		case "mode":
			status.Mode = string(value)
		case "keysize":
			idx := bytes.IndexByte(value, ' ')
			if idx >= 0 {
				status.Keysize, err = strconv.Atoi(string(value[:idx]))
			}
		case "offset":
			idx := bytes.IndexByte(value, ' ')
			if idx >= 0 {
				status.Offset, err = strconv.ParseInt(string(value[:idx]), 10, 64)
			}
		case "size":
			idx := bytes.IndexByte(value, ' ')
			if idx >= 0 {
				status.Size, err = strconv.ParseInt(string(value[:idx]), 10, 64)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return status, nil
}
