package aws

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

var sess *session.Session
var metaData *ec2metadata.EC2Metadata

var _idDoc *ec2metadata.EC2InstanceIdentityDocument
var _onceIDDoc sync.Once

var _ec2Client ec2iface.EC2API
var _onceEc2 sync.Once

func init() {
	sess = session.Must(session.NewSession())
	metaData = ec2metadata.New(sess)
}

func getIDDoc() (*ec2metadata.EC2InstanceIdentityDocument, error) {
	var err error
	_onceIDDoc.Do(func() {
		// Allow mock ID documents to be inserted
		if _idDoc == nil {
			var instance ec2metadata.EC2InstanceIdentityDocument
			instance, err = metaData.GetInstanceIdentityDocument()
			if err != nil {
				return
			}
			// Cache the document
			_idDoc = &instance
		}
	})
	return _idDoc, err
}

// Allocate a new EC2 client configured for the current instance
// region. Clients are re-used across multiple calls
func newEC2() (ec2iface.EC2API, error) {
	var err error
	_onceEc2.Do(func() {
		var id *ec2metadata.EC2InstanceIdentityDocument
		id, err = getIDDoc()
		if err != nil {
			return
		}
		if _ec2Client == nil {
			// Use the sess object already defined
			_ec2Client = ec2.New(sess, aws.NewConfig().WithRegion(id.Region))
		}
	})
	return _ec2Client, err
}
