package aws

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"time"
)

type awsclient struct {
	sess     *session.Session
	metaData *ec2metadata.EC2Metadata

	idDoc     *ec2metadata.EC2InstanceIdentityDocument
	onceIDDoc sync.Once

	ec2Client ec2iface.EC2API
	onceEc2   sync.Once
}

type combinedClient struct {
	*subnetsCacheClient
	*awsclient
	*interfaceClient
	*allocateClient
	*vpcCacheClient
}

// Client offers all of the supporting AWS services
type Client interface {
	InterfaceClient
	LimitsClient
	MetadataClient
	SubnetsClient
	AllocateClient
	VPCClient
}

var defaultClient *combinedClient

// DefaultClient that is setup with known defaults
var DefaultClient Client

func init() {
	awsClient := &awsclient{}
	subnets := &subnetsCacheClient{
		&subnetsClient{aws: awsClient},
		1 * time.Minute,
	}
	defaultClient = &combinedClient{
		subnets,
		awsClient,
		&interfaceClient{awsClient, subnets},
		&allocateClient{awsClient, subnets},
		&vpcCacheClient{
			&vpcclient{awsClient},
			1 * time.Hour,
		},
	}

	DefaultClient = defaultClient
	defaultClient.sess = session.Must(session.NewSession())
	defaultClient.metaData = ec2metadata.New(defaultClient.sess)
}

func (c *awsclient) getIDDoc() (*ec2metadata.EC2InstanceIdentityDocument, error) {
	var err error
	c.onceIDDoc.Do(func() {
		// Allow mock ID documents to be inserted
		if c.idDoc == nil {
			var instance ec2metadata.EC2InstanceIdentityDocument
			instance, err = c.metaData.GetInstanceIdentityDocument()
			if err != nil {
				return
			}
			// Cache the document
			c.idDoc = &instance
		}
	})
	return c.idDoc, err
}

// Allocate a new EC2 client configured for the current instance
// region. Clients are re-used across multiple calls
func (c *awsclient) newEC2() (ec2iface.EC2API, error) {
	var err error
	c.onceEc2.Do(func() {
		var id *ec2metadata.EC2InstanceIdentityDocument
		id, err = c.getIDDoc()
		if err != nil {
			return
		}
		if c.ec2Client == nil {
			// Use the sess object already defined
			c.ec2Client = ec2.New(c.sess, aws.NewConfig().WithRegion(id.Region))
		}
	})
	return c.ec2Client, err
}
