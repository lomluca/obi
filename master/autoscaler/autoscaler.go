package autoscaler

import (
	"github.com/sirupsen/logrus"
	"obi/master/model"
	"obi/master/utils"
	"time"
)

// The autoscaler module resize the managed cluster according to the policy.
// The policy is a pluggable struct with a well-defined interface to implement.
type Autoscaler struct {
	Policy Policy
	Timeout int16
	quit chan struct{}
	managedCluster model.Scalable
	allowDownscale bool
}

// Policy defines the primitive methods that must be implemented for any type of autoscaling policy
type Policy interface {
	Apply(*utils.ConcurrentSlice) int32
}

// New is the constructor of Autoscaler struct
// @param policy is the to apply for the autoscaling logic
// @param timeout is the time interval to wait before triggering the scaling-check action again
// @param cluster is the scalable cluster to be managed
// @param downscalePermitted is a bool to allow the policy to downscale
// return the pointer to the instance
func New(
	policy Policy,
	timeout int16,
	cluster model.Scalable,
	downscalePermitted bool,
	) *Autoscaler {
	return &Autoscaler{
		policy,
		timeout,
		make(chan struct{}),
		cluster,
		downscalePermitted,
	}
}


// StartMonitoring starts the execution of the autoscaler
func (as *Autoscaler) StartMonitoring() {
	go autoscalerRoutine(as)
}

// StopMonitoring stops the execution of the autoscaler
func (as *Autoscaler) StopMonitoring() {
	close(as.quit)
}

// goroutine which apply the scaling policy at each time interval. It will be stop when an empty object is inserted in
// the `quit` channel
// @param as is the autoscaler
func autoscalerRoutine(as *Autoscaler) {
	var delta int32
	for {
		select {
		case <-as.quit:
			logrus.WithField("clusterName", as.managedCluster.(model.ClusterBaseInterface).GetName()).Info(
				"Closing autoscaler routine.")
			return
		default:
			delta = as.Policy.Apply(as.managedCluster.(model.ClusterBaseInterface).GetMetricsWindow())

			if (delta < 0 && as.allowDownscale) || delta > 0 {
				as.managedCluster.Scale(delta)
			}
			time.Sleep(time.Duration(as.Timeout) * time.Second)
		}
	}
}
