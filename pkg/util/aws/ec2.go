package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/karlseguin/ccache"
	"github.com/prometheus/client_golang/prometheus"
)

// EC2 is our extension to AWS's ec2.EC2
type EC2 struct {
	ec2iface.EC2API
	cache APICache
}

// NewEC2 returns an awsutil EC2 service
func NewEC2(awsSession *session.Session) *EC2 {
	elbClient := EC2{
		ec2.New(awsSession),
		APICache{ccache.New(ccache.Configure())},
	}
	return &elbClient
}

// GetVPCID retrieves the VPC that the subents passed are contained in.
func (e *EC2) GetVPCID(subnets []*string) (*string, error) {
	var vpc *string

	if len(subnets) == 0 {
		return nil, fmt.Errorf("Empty subnet list provided to getVPCID")
	}

	key := fmt.Sprintf("%s-vpc", *subnets[0])
	item := e.cache.Get(key)

	if item == nil {
		subnetInfo, err := e.DescribeSubnets(&ec2.DescribeSubnetsInput{
			SubnetIds: subnets,
		})
		if err != nil {
			return nil, err
		}

		if len(subnetInfo.Subnets) == 0 {
			return nil, fmt.Errorf("DescribeSubnets returned no subnets")
		}

		vpc = subnetInfo.Subnets[0].VpcId
		e.cache.Set(key, vpc, time.Minute*60)

		AWSCache.With(prometheus.Labels{"cache": "vpc", "action": "miss"}).Add(float64(1))
	} else {
		vpc = item.Value().(*string)
		AWSCache.With(prometheus.Labels{"cache": "vpc", "action": "hit"}).Add(float64(1))
	}

	return vpc, nil
}
