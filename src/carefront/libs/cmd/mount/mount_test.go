package mount

import (
	"testing"
)

func TestParseLinuxMounts(t *testing.T) {
	mountOut := []byte(`/dev/xvda1 on / type ext4 (rw)
proc on /proc type proc (rw,noexec,nosuid,nodev)
sysfs on /sys type sysfs (rw,noexec,nosuid,nodev)
none on /sys/fs/cgroup type tmpfs (rw)
none on /sys/fs/fuse/connections type fusectl (rw)
none on /sys/kernel/debug type debugfs (rw)
none on /sys/kernel/security type securityfs (rw)
udev on /dev type devtmpfs (rw,mode=0755)
devpts on /dev/pts type devpts (rw,noexec,nosuid,gid=5,mode=0620)
tmpfs on /run type tmpfs (rw,noexec,nosuid,size=10%,mode=0755)
none on /run/lock type tmpfs (rw,noexec,nosuid,nodev,size=5242880)
none on /run/shm type tmpfs (rw,nosuid,nodev)
none on /run/user type tmpfs (rw,noexec,nosuid,nodev,size=104857600,mode=0755)
none on /sys/fs/pstore type pstore (rw)
systemd on /sys/fs/cgroup/systemd type cgroup (rw,noexec,nosuid,nodev,none,name=systemd)
/dev/xvdb on /mnt type ext3 (rw,_netdev)
/dev/mapper/mysql-data-encrypted on /mysql-data type xfs (rw) [mysql-data]
`)
	mounts, err := parseLinuxMounts(mountOut)
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 17 {
		t.Fatalf("Expected 17 mounts instead of %d", len(mounts))
	}
	if m := mounts["/mysql-data"]; m == nil {
		t.Fatal("Mount with label not found")
	} else if m.Device != "/dev/mapper/mysql-data-encrypted" {
		t.Fatal("Mount with label has invalid device")
	} else if m.Label != "mysql-data" {
		t.Fatal("Mount with label missing or wrong label")
	} else if m.Type != "xfs" {
		t.Fatal("Mount with label has wrong FS type")
	} else if _, ok := m.Options["rw"]; !ok {
		t.Fatal("Mount with lavel failed to parse option without value")
	}
	if m := mounts["/run/user"]; m == nil {
		t.Fatal("Mount /run/user missing")
	} else if m.Options["size"] != "104857600" {
		t.Fatal("Mount /run/user failed to parse options")
	}
}

func TestParseDarwinMounts(t *testing.T) {
	mountOut := []byte(`/dev/disk1 on / (hfs, local, journaled)
devfs on /dev (devfs, local, nobrowse)
map -hosts on /net (autofs, nosuid, automounted, nobrowse)
map auto_home on /home (autofs, automounted, nobrowse)
`)
	mounts, err := parseDarwinMounts(mountOut)
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 4 {
		t.Fatalf("Expected 4 mounts instead of %d", len(mounts))
	}
	if m := mounts["/"]; m == nil {
		t.Fatal("Mount / not found")
	} else if m.Device != "/dev/disk1" {
		t.Fatalf("Wrong device %s", m.Device)
	} else if m.Type != "hfs" {
		t.Fatalf("Write type %s", m.Type)
	} else if _, ok := m.Options["journaled"]; !ok {
		t.Fatalf("Option 'journaled' not found: %+v", m.Options)
	}
}
