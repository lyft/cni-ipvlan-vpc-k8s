package aws

// ENILimit contains limits for adapter count and addresses
type ENILimit struct {
	Adapters int
	IPv4     int
	IPv6     int
}

// LimitsClient provides methods for locating limits in AWS
type LimitsClient interface {
	ENILimits() ENILimit
}

var eniLimits map[string]ENILimit

func init() {
	// This table of limits referenced from:
	// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html
	eniLimits = map[string]ENILimit{
		"c1.medium":    {2, 6, 0},
		"c1.xlarge":    {4, 15, 0},
		"c3.large":     {3, 10, 10},
		"c3.xlarge":    {4, 15, 15},
		"c3.2xlarge":   {4, 15, 15},
		"c3.4xlarge":   {8, 30, 30},
		"c3.8xlarge":   {8, 30, 30},
		"c4.large":     {3, 10, 10},
		"c4.xlarge":    {4, 15, 15},
		"c4.2xlarge":   {4, 15, 15},
		"c4.4xlarge":   {8, 30, 30},
		"c4.8xlarge":   {8, 30, 30},
		"c5.large":     {3, 10, 10},
		"c5d.large":    {3, 10, 10},
		"c5.xlarge":    {4, 15, 15},
		"c5d.xlarge":   {4, 15, 15},
		"c5.2xlarge":   {4, 15, 15},
		"c5d.2xlarge":  {4, 15, 15},
		"c5.4xlarge":   {8, 30, 30},
		"c5d.4xlarge":  {8, 30, 30},
		"c5.9xlarge":   {8, 30, 30},
		"c5d.9xlarge":  {8, 30, 30},
		"c5.18xlarge":  {15, 50, 50},
		"c5d.18xlarge": {15, 50, 50},
		"cc2.8xlarge":  {8, 30, 0},
		"cg1.4xlarge":  {8, 30, 0},
		"cr1.8xlarge":  {8, 30, 0},
		"d2.xlarge":    {4, 15, 15},
		"d2.2xlarge":   {4, 15, 15},
		"d2.4xlarge":   {8, 30, 30},
		"d2.8xlarge":   {8, 30, 30},
		"f1.2xlarge":   {4, 15, 15},
		"f1.16xlarge":  {8, 50, 50},
		"g2.2xlarge":   {4, 15, 0},
		"g2.8xlarge":   {8, 30, 0},
		"g3.4xlarge":   {8, 30, 30},
		"g3.8xlarge":   {8, 30, 30},
		"g3.16xlarge":  {15, 50, 50},
		"hi1.4xlarge":  {8, 30, 0},
		"hs1.8xlarge":  {8, 30, 0},
		"i2.xlarge":    {4, 15, 15},
		"i2.2xlarge":   {4, 15, 15},
		"i2.4xlarge":   {8, 30, 30},
		"i2.8xlarge":   {8, 30, 30},
		"i3.large":     {3, 10, 10},
		"i3.xlarge":    {4, 15, 15},
		"i3.2xlarge":   {4, 15, 15},
		"i3.4xlarge":   {8, 30, 30},
		"i3.8xlarge":   {8, 30, 30},
		"i3.16xlarge":  {15, 50, 50},
		"m1.small":     {2, 4, 0},
		"m1.medium":    {2, 6, 0},
		"m1.large":     {3, 10, 0},
		"m1.xlarge":    {4, 15, 0},
		"m2.xlarge":    {4, 15, 0},
		"m2.2xlarge":   {4, 30, 0},
		"m2.4xlarge":   {8, 30, 0},
		"m3.medium":    {2, 6, 0},
		"m3.large":     {3, 10, 0},
		"m3.xlarge":    {4, 15, 0},
		"m3.2xlarge":   {4, 30, 0},
		"m4.large":     {2, 10, 10},
		"m4.xlarge":    {4, 15, 15},
		"m4.2xlarge":   {4, 15, 15},
		"m4.4xlarge":   {8, 30, 30},
		"m4.10xlarge":  {8, 30, 30},
		"m4.16xlarge":  {8, 30, 30},
		"m5.large":     {3, 10, 10},
		"m5d.large":    {3, 10, 10},
		"m5.xlarge":    {4, 15, 15},
		"m5d.xlarge":   {4, 15, 15},
		"m5.2xlarge":   {4, 15, 15},
		"m5d.2xlarge":  {4, 15, 15},
		"m5.4xlarge":   {8, 30, 30},
		"m5d.4xlarge":  {8, 30, 30},
		"m5.12xlarge":  {8, 30, 30},
		"m5d.12xlarge": {8, 30, 30},
		"m5.24xlarge":  {15, 50, 50},
		"m5d.24xlarge": {15, 50, 50},
		"p2.xlarge":    {4, 15, 15},
		"p2.8xlarge":   {8, 30, 30},
		"p2.16xlarge":  {8, 30, 30},
		"p3.2xlarge":   {4, 15, 15},
		"p3.8xlarge":   {8, 30, 30},
		"p3.16xlarge":  {8, 30, 30},
		"r3.large":     {3, 10, 10},
		"r3.xlarge":    {4, 15, 15},
		"r3.2xlarge":   {4, 15, 15},
		"r3.4xlarge":   {8, 30, 30},
		"r3.8xlarge":   {8, 30, 30},
		"r4.large":     {3, 10, 10},
		"r4.xlarge":    {4, 15, 15},
		"r4.2xlarge":   {4, 15, 15},
		"r4.4xlarge":   {8, 30, 30},
		"r4.8xlarge":   {8, 30, 30},
		"r4.16xlarge":  {15, 50, 50},
		"t1.micro":     {2, 2, 0},
		"t2.nano":      {2, 2, 2},
		"t2.micro":     {2, 2, 2},
		"t2.small":     {2, 4, 4},
		"t2.medium":    {3, 6, 6},
		"t2.large":     {3, 12, 12},
		"t2.xlarge":    {3, 15, 15},
		"t2.2xlarge":   {3, 15, 15},
		"x1.16xlarge":  {8, 30, 30},
		"x1.32xlarge":  {8, 30, 30},
		"x1e.32xlarge": {8, 30, 30},
	}
}

// ENILimitsForInstanceType returns the limits for ENI for an instance type
// Returns a zero-limit for unknown instance types
func ENILimitsForInstanceType(itype string) (limit ENILimit) {
	limit = eniLimits[itype]
	return
}

// ENILimits returns the limits based on the system's instance type
func (c *awsclient) ENILimits() ENILimit {
	id, err := c.getIDDoc()
	if err != nil || id == nil {
		return ENILimit{}
	}
	return ENILimitsForInstanceType(id.InstanceType)
}
