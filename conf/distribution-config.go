package conf

type DistributeConfig struct {
	// DistributionType used to tag witch method to distribute tax,0 means all tax feed to orderer which deal the order
	// 1 means tax average to all orderer
	DistributionType int `json:"allocation_rule"`
}

func (d *DistributeConfig) String() string {
	if d.DistributionType == 0 {
		return "effort-based allocation"
	} else {
		return "equal allocation"
	}
}

type SortConfig struct {
	SortSwitch bool `json:"sort_switch"`
}

func (d *SortConfig) String() string {
	if d.SortSwitch {
		return "sorted by transaction fees"
	} else {
		return "sorted by the timestamps"
	}
}

type OrdererCapacityConfig struct {
	Orderer  string `json:"orderer"`
	Capacity int    `json:"capacity"`
}

type Feedback struct {
	Orderer   string `json:"orderer"`
	Capacity  int    `json:"capacity"`
	FeeReward string `json:"fee_reward"`
}

type OrdererFeedback struct {
	Lists []*Feedback `json:"lists"`
}

type TransactionFee struct {
	Fee int `json:"fee"`
}
