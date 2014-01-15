package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"

	"carefront/libs/aws"
	"carefront/libs/aws/ec2"
	"carefront/libs/cmd/cryptsetup"
)

var config = struct {
	Bastion     string
	Environment string
	AWSRole     string
	AZ          string
	User        string
	StripeSize  int
	Readahead   int
	Iops        int
	Cipher      string
	Verbose     bool

	awsAuth aws.Auth
	ec2     *ec2.EC2
}{
	User:       os.Getenv("USER"),
	StripeSize: 4, // KB
	Cipher:     cryptsetup.DefaultCipher,
}

func init() {
	flag.StringVar(&config.Bastion, "bastion", config.Bastion, "SSH bastion host")
	flag.StringVar(&config.AZ, "az", config.AZ, "Availability zone")
	flag.StringVar(&config.Environment, "env", config.Environment, "Environment")
	flag.StringVar(&config.User, "user", config.User, "User for SSH")
	flag.StringVar(&config.AWSRole, "role", config.AWSRole, "AWS Role")
	flag.IntVar(&config.Iops, "iops", config.Iops, "Provisioned IOPS (0=disable)")
	flag.IntVar(&config.StripeSize, "stripesize", config.StripeSize, "Stripe size in KB")
	flag.BoolVar(&config.Verbose, "v", config.Verbose, "Verbose output")
}

func main() {
	flag.Parse()
	if config.Environment == "" {
		fmt.Fprintf(os.Stderr, "-env is required\n")
		os.Exit(1)
	}

	if config.AWSRole != "" {
		if config.AWSRole == "*" {
			config.AWSRole = ""
		}
		cred, err := aws.CredentialsForRole(config.AWSRole)
		if err != nil {
			log.Fatal(err)
		}
		config.awsAuth = cred
	} else {
		if keys := aws.KeysFromEnvironment(); keys.AccessKey == "" || keys.SecretKey == "" {
			if cred, err := aws.CredentialsForRole(""); err == nil {
				config.awsAuth = cred
			} else {
				log.Fatal("Missing AWS_ACCESS_KEY or AWS_SECRET_KEY")
			}
		} else {
			config.awsAuth = keys
		}
	}

	if config.AZ == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			log.Fatalf("no region specified and failed to get from instance metadata: %+v", err)
		}
		config.AZ = az
	}

	config.ec2 = &ec2.EC2{
		Region: aws.Regions[config.AZ[:len(config.AZ)-1]],
		Client: &aws.Client{Auth: config.awsAuth},
	}

	var err error
	switch flag.Arg(0) {
	default:
		err = fmt.Errorf("Unknown command %s", flag.Arg(0))
	case "":
		err = fmt.Errorf("Commands: create, attach")
	case "create":
		err = create()
	case "attach":
		err = attach()
	case "detach":
		err = detach()
	case "luksmount":
		err = luksMount()
	case "gcsnapshots":
		err = gcSnapshots()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func findGroup(name string) ([]*ec2.Volume, error) {
	return config.ec2.DescribeVolumes(nil, map[string][]string{
		"availability-zone": []string{config.AZ},
		"tag:Group":         []string{name},
		"tag:Environment":   []string{config.Environment},
	})
}

type snapshotSort []*ec2.Snapshot

func (s snapshotSort) Len() int {
	return len(s)
}

func (s snapshotSort) Less(a, b int) bool {
	return s[a].Description >= s[b].Description
}

func (s snapshotSort) Swap(a, b int) {
	t := s[a]
	s[a] = s[b]
	s[b] = t
}

func create() error {
	if len(flag.Args()) < 4 {
		return fmt.Errorf("usage: create [name] [size] [stripes or snapshotgroupname] [snapshot-description]")
	}
	name := flag.Arg(1)
	size, err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		return err
	}
	snapshotGroupName := ""
	count, err := strconv.Atoi(flag.Arg(3))
	if err != nil {
		snapshotGroupName = flag.Arg(3)
	}

	snapshotDescription := ""
	if len(flag.Args()) > 4 {
		snapshotDescription = flag.Arg(4)
	}

	if snapshotGroupName == "" && snapshotDescription != "" {
		return fmt.Errorf("Cannot have snapshot description specified as %s when there is no snapshot group name specified.", snapshotDescription)
	}

	if vols, err := findGroup(name); err != nil {
		return err
	} else if len(vols) != 0 {
		return fmt.Errorf("Group %s already exists", name)
	}

	var snapshots []*ec2.Snapshot // Snapshot IDs
	if snapshotGroupName != "" {

		filters := map[string][]string{
			"tag:Group":       []string{snapshotGroupName},
			"tag:Environment": []string{config.Environment},
		}

		if snapshotDescription != "" {
			filters["description"] = []string{snapshotDescription}
		}

		snaps, err := config.ec2.DescribeSnapshots(nil, []string{"self"}, nil, filters)
		if err != nil {
			return fmt.Errorf("Failed to lookup snapshots: %+v", err)
		}
		if len(snaps) == 0 {
			if snapshotDescription == "" {
				return fmt.Errorf("No snapshots found for group %s", snapshotGroupName)
			}
			return fmt.Errorf("No snapshots found for group %s with description %s", snapshotGroupName, snapshotDescription)
		}
		sort.Sort(snapshotSort(snaps))

		s := snaps[0]
		desc := s.Description
		count, err = strconv.Atoi(s.Tags["Total"])
		if err != nil {
			return err
		}
		snapshots = make([]*ec2.Snapshot, count)

		for _, s := range snaps[:count] {
			if s.Description != desc {
				return fmt.Errorf("Snapshot group not complete: %s", desc)
			}
			num, err := strconv.Atoi(s.Tags["Number"])
			if err != nil {
				return err
			}
			snapshots[num-1] = s
		}
	}

	for i := 0; i < count; i++ {
		snap := ""
		if len(snapshots) != 0 {
			snap = snapshots[i].SnapshotId
		}

		vol, err := config.ec2.CreateVolume(size, config.AZ, "", snap, config.Iops)
		if err != nil {
			return err
		}
		tags := map[string]string{
			"Name":        fmt.Sprintf("%s-%s-%d", config.Environment, name, i+1),
			"Group":       name,
			"Number":      strconv.Itoa(i + 1),
			"Environment": config.Environment,
			"Total":       strconv.Itoa(count),
		}
		fmt.Printf("Created volume %s (%s)\n", tags["Name"], vol.VolumeId)
		if snapshotGroupName != "" {
			tags["SnapshotGroup"] = snapshotGroupName
		}
		if err := config.ec2.CreateTags([]string{vol.VolumeId}, tags); err != nil {
			log.Printf("Failed to create tags for %s", vol.VolumeId)
		}
	}

	return nil
}

func attach() error {
	if len(flag.Args()) < 3 {
		return fmt.Errorf("usage: attach [name] [firstdevice] <instanceid>")
	}
	name := flag.Arg(1)
	firstDevice := flag.Arg(2)
	instanceId := flag.Arg(3)
	if instanceId == "" {
		if id, err := aws.GetMetadata(aws.MetadataInstanceID); err != nil {
			return fmt.Errorf("Instance ID required when not running on EC2")
		} else {
			instanceId = id
		}
	}

	vols, err := findGroup(name)
	if err != nil {
		return err
	}
	if len(vols) == 0 {
		return fmt.Errorf("Group %s does not exist", name)
	}

	// Validate the correct number of volumes were returned
	if total, err := strconv.Atoi(vols[0].Tags["Total"]); err != nil {
		return err
	} else if len(vols) != total {
		return fmt.Errorf("Expected %d volumes but found %d", total, len(vols))
	}

	// Make sure the volumes aren't already attached
	for _, v := range vols {
		if v.Attachment != nil {
			return fmt.Errorf("Volume %s (%s) is already attached to %s (%s)", v.VolumeId, v.Tags["Name"], v.Attachment.InstanceId, v.Attachment.Status)
		}
	}

	for _, v := range vols {
		num, err := strconv.Atoi(v.Tags["Number"])
		if err != nil {
			return err
		}
		dev := firstDevice[:len(firstDevice)-1] + string(firstDevice[len(firstDevice)-1]+uint8(num-1))
		fmt.Printf("Attaching %s (%s) to %s as %s... ", v.VolumeId, v.Tags["Name"], instanceId, dev)
		if res, err := config.ec2.AttachVolume(v.VolumeId, instanceId, dev); err != nil {
			return err
		} else {
			fmt.Printf("%s\n", res.Status)
		}
	}
	return nil
}

func detach() error {
	if len(flag.Args()) < 2 {
		return fmt.Errorf("usage: detach [name]")
	}
	name := flag.Arg(1)

	vols, err := findGroup(name)
	if err != nil {
		return err
	}
	if len(vols) == 0 {
		return fmt.Errorf("Group %s does not exist", name)
	}

	for _, v := range vols {
		if v.Attachment != nil && v.Attachment.Status != "available" {
			fmt.Printf("Detaching %s (%s) from %s... ", v.VolumeId, v.Tags["Name"], v.Attachment.InstanceId)
			if res, err := config.ec2.DetachVolume(v.VolumeId, "", "", false); err != nil {
				return err
			} else {
				fmt.Println(res.Status)
			}
		}
	}

	return nil
}

func gcSnapshots() error {
	if len(flag.Args()) < 3 {
		return fmt.Errorf("usage: gcsnapshots [name] [#tokeep]")
	}
	name := flag.Arg(1)
	toKeep, err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		return err
	}
	if toKeep < 0 {
		toKeep = 0
	}

	snaps, err := config.ec2.DescribeSnapshots(nil, []string{"self"}, nil,
		map[string][]string{
			"tag:Group":       []string{name},
			"tag:Environment": []string{config.Environment},
		})
	if err != nil {
		return fmt.Errorf("Failed to lookup snapshots: %+v", err)
	}
	sort.Sort(snapshotSort(snaps))

	// s := snaps[0]
	// desc := s.Description
	// count, err = strconv.Atoi(s.Tags["Total"])
	// if err != nil {
	// 	return err
	// }
	// snapshots = make([]*ec2.Snapshot, count)

	desc := ""
	for _, s := range snaps {
		if s.Description != desc {
			first := desc == ""
			desc = s.Description
			if !first {
				toKeep--
			}
			if toKeep <= 0 && config.Verbose {
				fmt.Printf("Deleting snapshot group '%s'\n", s.Description)
			}
		}
		if toKeep <= 0 {
			if err := config.ec2.DeleteSnapshot(s.SnapshotId); err != nil {
				return err
			}
		}
	}
	// if toKeep > 1 {
	// 	toKeep = 1
	// }
	// if config.Verbose {
	// 	fmt.Printf("Deleted %d volume group snapshots\n", -toKeep+1)
	// }

	return nil
}
