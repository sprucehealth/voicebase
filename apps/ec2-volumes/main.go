package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/cmd/cryptsetup"
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

	awsConfig *aws.Config
	ec2       *ec2.EC2
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
	log.SetFlags(0)

	flag.Parse()
	if config.Environment == "" {
		fmt.Fprintf(os.Stderr, "-env is required\n")
		os.Exit(1)
	}

	creds := credentials.NewEnvCredentials()
	if c, err := creds.Get(); err != nil || c.AccessKeyID == "" || c.SecretAccessKey == "" {
		creds = ec2rolecreds.NewCredentials(ec2metadata.New(&ec2metadata.Config{
			HTTPClient: &http.Client{Timeout: 2 * time.Second},
		}), time.Minute*10)
	}
	if config.AZ == "" {
		az, err := awsutil.GetMetadata(awsutil.MetadataAvailabilityZone)
		if err != nil {
			log.Fatalf("no region specified and failed to get from instance metadata: %+v", err)
		}
		config.AZ = az
	}
	config.awsConfig = &aws.Config{
		Credentials: creds,
		Region:      aws.String(config.AZ[:len(config.AZ)-1]),
	}
	config.ec2 = ec2.New(config.awsConfig)

	var err error
	switch flag.Arg(0) {
	default:
		err = fmt.Errorf("unknown command %s", flag.Arg(0))
	case "":
		err = fmt.Errorf("commands: create, attach")
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
	res, err := config.ec2.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("availability-zone"), Values: []*string{&config.AZ}},
			{Name: aws.String("tag:Group"), Values: []*string{&name}},
			{Name: aws.String("tag:Environment"), Values: []*string{&config.Environment}},
		},
	})
	if err != nil {
		return nil, err
	}
	return res.Volumes, nil
}

type snapshotSort []*ec2.Snapshot

func (s snapshotSort) Len() int {
	return len(s)
}

func (s snapshotSort) Less(a, b int) bool {
	return *s[a].Description >= *s[b].Description
}

func (s snapshotSort) Swap(a, b int) {
	t := s[a]
	s[a] = s[b]
	s[b] = t
}

func create() error {
	if len(flag.Args()) < 4 {
		return fmt.Errorf("usage: create [name] [size] [stripes or snapshotgroupname] [optional: snapshot-description]")
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
		return fmt.Errorf("cannot have snapshot description specified as %s when there is no snapshot group name specified", snapshotDescription)
	}

	if vols, err := findGroup(name); err != nil {
		return err
	} else if len(vols) != 0 {
		return fmt.Errorf("group %s already exists", name)
	}

	var snapshots []*ec2.Snapshot // Snapshot IDs
	if snapshotGroupName != "" {
		filters := []*ec2.Filter{
			{Name: aws.String("tag:Group"), Values: []*string{&snapshotGroupName}},
			{Name: aws.String("tag:Environment"), Values: []*string{&config.Environment}},
		}

		if snapshotDescription != "" {
			filters = append(filters, &ec2.Filter{Name: aws.String("description"), Values: []*string{&snapshotDescription}})
		}

		res, err := config.ec2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
			OwnerIds: []*string{aws.String("self")},
			Filters:  filters,
		})
		if err != nil {
			return fmt.Errorf("failed to lookup snapshots: %+v", err)
		}
		if len(res.Snapshots) == 0 {
			if snapshotDescription == "" {
				return fmt.Errorf("no snapshots found for group %s", snapshotGroupName)
			}
			return fmt.Errorf("no snapshots found for group %s with description %s", snapshotGroupName, snapshotDescription)
		}
		sort.Sort(snapshotSort(res.Snapshots))

		s := res.Snapshots[0]
		desc := s.Description
		count, err = strconv.Atoi(tag(s.Tags, "Total"))
		if err != nil {
			return err
		}
		snapshots = make([]*ec2.Snapshot, count)

		for _, s := range res.Snapshots[:count] {
			if *s.Description != *desc {
				return fmt.Errorf("snapshot group not complete: %s", *desc)
			}
			num, err := strconv.Atoi(tag(s.Tags, "Number"))
			if err != nil {
				return err
			}
			snapshots[num-1] = s
		}
	}

	for i := 0; i < count; i++ {
		snap := ""
		if len(snapshots) != 0 {
			snap = *snapshots[i].SnapshotId
		}

		vol, err := config.ec2.CreateVolume(&ec2.CreateVolumeInput{
			Size:             aws.Int64(int64(size)),
			AvailabilityZone: &config.AZ,
			SnapshotId:       &snap,
			Iops:             aws.Int64(int64(config.Iops)),
		})
		if err != nil {
			return err
		}
		tags := []*ec2.Tag{
			{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%s-%d", config.Environment, name, i+1))},
			{Key: aws.String("Group"), Value: &name},
			{Key: aws.String("Number"), Value: aws.String(strconv.Itoa(i + 1))},
			{Key: aws.String("Environment"), Value: &config.Environment},
			{Key: aws.String("Total"), Value: aws.String(strconv.Itoa(count))},
		}
		fmt.Printf("Created volume %s (%s)\n", tag(tags, "Name"), *vol.VolumeId)
		if snapshotGroupName != "" {
			tags = append(tags, &ec2.Tag{Key: aws.String("SnapshotGroup"), Value: &snapshotGroupName})
		}
		if _, err := config.ec2.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{vol.VolumeId},
			Tags:      tags,
		}); err != nil {
			log.Printf("Failed to create tags for %s", *vol.VolumeId)
		}
	}

	return nil
}

func attach() error {
	if len(flag.Args()) < 3 {
		return fmt.Errorf("usage: attach [name] [firstdevice] <instanceID>")
	}
	name := flag.Arg(1)
	firstDevice := flag.Arg(2)
	instanceID := flag.Arg(3)
	if instanceID == "" {
		var err error
		instanceID, err = awsutil.GetMetadata(awsutil.MetadataInstanceID)
		if err != nil {
			return fmt.Errorf("instance ID required when not running on EC2")
		}
	}

	// If the instanceID doesn't look like an instance_id (starting with "i-)
	// then see if there's an instance with the tag Name that matches.
	if instanceID[:2] != "i-" {
		res, err := config.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("tag:Name"), Values: []*string{&instanceID}},
				{Name: aws.String("tag:Environment"), Values: []*string{&config.Environment}},
			},
		})
		if err != nil {
			return err
		}
		if n := len(res.Reservations); n == 0 {
			return fmt.Errorf("instance with name %s not found", instanceID)
		} else if n > 1 {
			return fmt.Errorf("more than one reservation (%d) with name %s not found", n, instanceID)
		}
		if n := len(res.Reservations[0].Instances); n > 1 {
			return fmt.Errorf("more than one instance (%d) with name %s not found", n, instanceID)
		}
		instanceID = *res.Reservations[0].Instances[0].InstanceId
	}

	vols, err := findGroup(name)
	if err != nil {
		return err
	}
	if len(vols) == 0 {
		return fmt.Errorf("group %s does not exist", name)
	}

	// Validate the correct number of volumes were returned
	if total, err := strconv.Atoi(tag(vols[0].Tags, "Total")); err != nil {
		return err
	} else if len(vols) != total {
		return fmt.Errorf("expected %d volumes but found %d", total, len(vols))
	}

	// Make sure the volumes aren't already attached
	for _, v := range vols {
		if len(v.Attachments) != 0 {
			return fmt.Errorf("volume %s (%s) is already attached to %s (%s)", *v.VolumeId, tag(v.Tags, "Name"), *v.Attachments[0].InstanceId, *v.Attachments[0].State)
		}
	}

	for _, v := range vols {
		num, err := strconv.Atoi(tag(v.Tags, "Number"))
		if err != nil {
			return err
		}
		dev := firstDevice[:len(firstDevice)-1] + string(firstDevice[len(firstDevice)-1]+uint8(num-1))
		fmt.Printf("Attaching %s (%s) to %s as %s... ", *v.VolumeId, tag(v.Tags, "Name"), instanceID, dev)
		res, err := config.ec2.AttachVolume(&ec2.AttachVolumeInput{
			VolumeId:   v.VolumeId,
			InstanceId: &instanceID,
			Device:     &dev,
		})
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", *res.State)
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
		return fmt.Errorf("group %s does not exist", name)
	}

	for _, v := range vols {
		if len(v.Attachments) != 0 && *v.Attachments[0].State != "available" {
			fmt.Printf("Detaching %s (%s) from %s... ", *v.VolumeId, tag(v.Tags, "Name"), *v.Attachments[0].InstanceId)
			res, err := config.ec2.DetachVolume(&ec2.DetachVolumeInput{VolumeId: v.VolumeId})
			if err != nil {
				return err
			}
			fmt.Println(res.State)
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

	res, err := config.ec2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{aws.String("self")},
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Group"), Values: []*string{&name}},
			{Name: aws.String("tag:Environment"), Values: []*string{&config.Environment}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to lookup snapshots: %+v", err)
	}
	sort.Sort(snapshotSort(res.Snapshots))

	// s := snaps[0]
	// desc := s.Description
	// count, err = strconv.Atoi(s.Tags["Total"])
	// if err != nil {
	// 	return err
	// }
	// snapshots = make([]*ec2.Snapshot, count)

	desc := ""
	for _, s := range res.Snapshots {
		if *s.Description != desc {
			first := desc == ""
			desc = *s.Description
			if !first {
				toKeep--
			}
			if toKeep <= 0 && config.Verbose {
				fmt.Printf("deleting snapshot group '%s'\n", *s.Description)
			}
		}
		if toKeep <= 0 {
			if _, err := config.ec2.DeleteSnapshot(&ec2.DeleteSnapshotInput{SnapshotId: s.SnapshotId}); err != nil {
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

func tag(tags []*ec2.Tag, key string) string {
	for _, t := range tags {
		if *t.Key == key {
			return *t.Value
		}
	}
	return ""
}
