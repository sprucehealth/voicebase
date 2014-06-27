package ec2

import (
	"fmt"
	"net/url"
)

func (ec2 *EC2) DescribeNetworkACLs(ids []string, filters map[string][]string) ([]*NetworkACL, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("NetworkAclId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeNetworkACLsResponse{}
	err := ec2.Get("DescribeNetworkAcls", params, res)
	if err != nil {
		return nil, err
	}
	return res.NetworkACLs, nil
}

func (ec2 *EC2) DescribeRouteTables(ids []string, filters map[string][]string) ([]*RouteTable, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("RouteTableId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeRouteTablesResponse{}
	err := ec2.Get("DescribeRouteTables", params, res)
	if err != nil {
		return nil, err
	}
	return res.RouteTables, nil
}

func (ec2 *EC2) DescribeSecurityGroups(names, ids []string, filters map[string][]string) ([]*SecurityGroup, error) {
	params := url.Values{}
	for i, n := range names {
		params.Set(fmt.Sprintf("GroupName.%d", i+1), n)
	}
	for i, id := range ids {
		params.Set(fmt.Sprintf("GroupId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeSecurityGroupsResponse{}
	err := ec2.Get("DescribeSecurityGroups", params, res)
	if err != nil {
		return nil, err
	}
	return res.SecurityGroups, nil
}

func (ec2 *EC2) DescribeSubnets(ids []string, filters map[string][]string) ([]*Subnet, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("SubnetId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeSubnetsResponse{}
	err := ec2.Get("DescribeSubnets", params, res)
	if err != nil {
		return nil, err
	}
	return res.Subnets, nil
}

func (ec2 *EC2) DescribeVPCs(ids []string, filters map[string][]string) ([]*VPC, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("vpcId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeVPCsResponse{}
	err := ec2.Get("DescribeVpcs", params, res)
	if err != nil {
		return nil, err
	}
	return res.VPCs, nil
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-DescribeVpcPeeringConnections.html
func (ec2 *EC2) DescribeVPCPeeringConnections(ids []string, filters map[string][]string) ([]*VPCPeeringConnection, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("VpcPeeringConnectionId.%d", i+1), id)
	}
	encodeFilters(params, filters)
	res := &DescribeVPCPeeringConnectionsResponse{}
	err := ec2.Get("DescribeVpcPeeringConnections", params, res)
	if err != nil {
		return nil, err
	}
	return res.Connections, nil
}
