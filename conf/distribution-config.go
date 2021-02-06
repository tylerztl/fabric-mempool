package conf

type DistributeConfig struct {
	// DistributionType used to tag witch method to distribute tax,0 means all tax feed to orderer which deal the order
	// 1 means tax average to all orderer
	DistributionType int `json:"distribution_type"`
}

func (d *DistributeConfig) String() string {
	if d.DistributionType == 0 {
		return "all to one"
	} else {
		return "average"
	}
}
