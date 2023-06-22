package lnd

type LNDCluster struct {
	members []*LNDClusterMember
}

type LNDClusterMember struct {
	client   LNDWrapper
	isOnline bool
}

func (cluster *LNDCluster) checkClusterStatus() {
	//for all nodes
	//- call getinfo
	//- if num_active_channels / num_total_channels < x % (50?)
	//- == offline
}

//make cluster implement interface
//no subscriber functionality
//loop over members, use first online member to make the payment
//start loop to check cluster status every 30s
