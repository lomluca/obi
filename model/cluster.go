package model

import "sync"

// Scalable is the interface that must be implemented from a scalable cluster
type Scalable interface {
	Scale(nodes int16, down bool)
	Status() Metrics
}

// ClusterBase is the base class for any type of cluster
type ClusterBase struct {
	Name string
	Nodes int16
	status Metrics // not available outside package to prevent race conditions- get and set must be used
	sync.Mutex
}


// NewClusterBase is the constructor of ClusterBase struct
// @param clusterName is the name of the cluster
// @param size is the number of nodes in the cluster
// return the pointer to the ClusterBase instance
func NewClusterBase(clusterName string, size int16) *ClusterBase {
	return &ClusterBase{
		Name:  clusterName,
		Nodes: size,
	}
}

// GetMetricsSnapshot is the getter of status field inside ClusterBase
// thread-safe
func (c *ClusterBase) GetMetricsSnapshot() Metrics {
	c.Lock()
	defer c.Unlock()

	value := c.status
	return value
}

// SetMetricsSnapshot is the setter of status field inside ClusterBase
// thread-safe
func (c *ClusterBase) SetMetricsSnapshot(newStatus Metrics) {
	c.Lock()
	defer c.Unlock()

	c.status = newStatus
}