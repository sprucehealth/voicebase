package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sprucehealth/backend/libs/cmd"
	"github.com/sprucehealth/backend/libs/cmd/cryptsetup"
	"github.com/sprucehealth/backend/libs/cmd/lvm"
	"github.com/sprucehealth/backend/libs/cmd/mount"
	"github.com/sprucehealth/backend/libs/cmd/xfs"
)

func luksMount() error {
	if len(flag.Args()) < 4 {
		return fmt.Errorf("usage: luksmount [name] [mountname] [keyfile]")
	}

	// Read key
	fi, err := os.Open(flag.Arg(3))
	if err != nil {
		return err
	}
	defer fi.Close()
	key, err := ioutil.ReadAll(fi)
	if err != nil {
		return err
	}

	name := flag.Arg(1)
	mountName := flag.Arg(2)

	vols, err := findGroup(name)
	if err != nil {
		return err
	}
	if len(vols) == 0 {
		return fmt.Errorf("goup %s does not exist", name)
	}

	// Validate the correct number of volumes were returned
	if total, err := strconv.Atoi(tag(vols[0].Tags, "Total")); err != nil {
		return err
	} else if len(vols) != total {
		return fmt.Errorf("expected %d volumes but found %d", total, len(vols))
	}

	// Make sure the volumes are attached to an instance
	instanceID := ""
	var devices []string
	for _, v := range vols {
		status := ""
		if len(v.Attachments) != 0 {
			status = *v.Attachments[0].State
		}
		if status != "attached" {
			return fmt.Errorf("volume %s (%s) is not attached: %s", *v.VolumeID, tag(v.Tags, "Name"), status)
		}
		if instanceID == "" {
			instanceID = *v.Attachments[0].InstanceID
		} else if instanceID != *v.Attachments[0].InstanceID {
			return fmt.Errorf("some volumes are attached to different instances")
		}
		devices = append(devices, *v.Attachments[0].Device)
	}
	sort.Strings(devices)

	res, err := config.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIDs: []*string{&instanceID},
	})
	if err != nil {
		return err
	}
	if len(res.Reservations) != 1 {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	inst := res.Reservations[0].Instances[0]
	ip := *inst.PrivateIPAddress
	fmt.Printf("IP: %s\n", ip)

	cmr, err := cmd.NewSSHCommander(fmt.Sprintf("%s@%s:22", config.User, ip), fmt.Sprintf("%s:22", config.Bastion))
	if err != nil {
		log.Fatal(err)
	}
	defer cmr.Close()

	lv := &lvm.LVM{Cmd: cmr}
	cs := &cryptsetup.Cryptsetup{Cmd: cmr}
	xf := &xfs.XFS{Cmd: cmr}
	mnt := &mount.MountCmd{Cmd: cmr}

	vgName := mountName + "-vg"
	lvName := mountName + "-lv"
	lvDev := fmt.Sprintf("/dev/%s/%s", vgName, lvName)
	encryptedName := mountName + "-encrypted"
	luksDev := "/dev/mapper/" + encryptedName
	mountPath := "/" + mountName

	// Vreate LVM volume group if necessary
	if vgs, err := lv.VGDisplay(); err != nil {
		return fmt.Errorf("VGDisplay failed: %s", err.Error())
	} else if vgs[vgName] == nil {
		fmt.Println("Creating physical volumes...")
		pvs, err := lv.PVDisplay()
		if err != nil {
			return err
		}
		for _, dev := range devices {
			if pvs[dev] != nil {
				return fmt.Errorf("device '%s' already is an LVM physical volume", dev)
			}
			fmt.Printf("Creating phyical volume on %s...\n", dev)
			if err := lv.PVCreate(dev); err != nil {
				return fmt.Errorf("pvcreate failed for device %s: %+v", dev, err)
			}
		}

		fmt.Printf("Creating volume group %s...\n", vgName)
		if err := lv.VGCreate(vgName, devices); err != nil {
			return fmt.Errorf("vgcreate failed for name %s devices %+v: %+v", vgName, devices, err)
		}
		fmt.Printf("Creating logical volume %s...\n", lvName)
		if err := lv.LVCreate(lvName, vgName, "100%FREE", len(devices), config.StripeSize, config.Readahead); err != nil {
			return fmt.Errorf("lvcreate failed: %+v", err)
		}
	} else {
		fmt.Printf("Volume group %s found\n", vgName)
	}

	// Check if formatted for LUKS
	if isLuks, err := cs.IsLuks(lvDev); err != nil {
		return fmt.Errorf("isLuks failed: %+v", err)
	} else if !isLuks {
		fmt.Println("Volume is not LUKS. Initializing...")
		if err := cs.LuksFormat(lvDev, config.Cipher, key); err != nil {
			return fmt.Errorf("luks format failed: %+v", err)
		}
	}

	// Try to open the LUKS device
	fmt.Println("Opening LUKS device...")
	if err := cs.LuksOpen(encryptedName, lvDev, key); err != nil {
		return fmt.Errorf("failed to open LUKS device: %+v", err)
	}

	if is, _, _, err := xf.IsXFS(luksDev); err != nil {
		return fmt.Errorf("failed to check for XFS: %+v", err)
	} else if !is {
		fmt.Println("Formatting LUKS dev as XFS...")
		if err := xf.Format(luksDev); err != nil {
			return fmt.Errorf("failed to format LUKS device as XFS: %+v", err)
		}
		if err := xf.SetLabel(luksDev, mountName); err != nil {
			fmt.Printf("failed to set label of %s to %s\n", luksDev, mountName)
		}
	}

	c, err := cmr.Command("sudo", "mkdir", "-p", mountPath)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to create mount path %s: %+v", mountPath, err)
	}

	fmt.Println("Mounting...")
	if err := mnt.Mount(luksDev, mountPath); err != mount.ErrAlreadyMounted && err != nil {
		return fmt.Errorf("failed to mount: %+v", err)
	}

	return nil
}

// def luks_unmount(name, luks_name):
//     vols = find_group(name)
//     if not vols:
//         fail("Group %s does not exist", name)

//     # Validate the correct number of volumes were returned
//     total = int(vols[0].tags["Total"])
//     if len(vols) != total:
//         fail("Expected %d volumes but found %d", total, len(vols))

//     # Make sure the volumes are attached to an instance
//     instance_id = None
//     for v in vols:
//         if v.attach_data.status == None:
//             fail("Volume %s (%s) isn't attached")
//         if instance_id is None:
//             instance_id = v.attach_data.instance_id
//         elif instance_id != v.attach_data.instance_id:
//             fail("Some volumes are attached to different instances")

//     inst = ec2.get_only_instances([instance_id])[0]
//     ip = inst.ip_address or inst.private_ip_address

//     encrypted_name = "%s-encrypted" % luks_name
//     subprocess.check_call(["ssh", "ubuntu@%s" % ip, "sudo umount /%s && sudo cryptsetup luksClose %s" % (luks_name, encrypted_name)])
