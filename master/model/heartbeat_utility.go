package model

import (
		"obi/master/predictor"
)

// MetricsToSnapshot converts an heartbeat to a snapshot of metrics to be sent to the predictor module
func MetricsToSnapshot(message Metrics) *predictor.MetricsSnasphot {
	return &predictor.MetricsSnasphot{
		AMResourceLimitMB: message.AMResourceLimitMB,
		AMResourceLimitVCores: message.AMResourceLimitVCores,
		UsedAMResourceMB: message.UsedAMResourceMB,
		UsedAMResourceVCores: message.UsedAMResourceVCores,
		AppsSubmitted: message.AppsSubmitted,
		AppsRunning: message.AppsRunning,
		AppsPending: message.AppsPending,
		AppsCompleted: message.AppsCompleted,
		AppsKilled: message.AppsKilled,
		AppsFailed: message.AppsFailed,
		AggregateContainersPreempted: message.AggregateContainersPreempted,
		ActiveApplications: message.ActiveApplications,
		AppAttemptFirstContainerAllocationDelayNumOps: message.AppAttemptFirstContainerAllocationDelayNumOps,
		AppAttemptFirstContainerAllocationDelayAvgTime: message.AppAttemptFirstContainerAllocationDelayAvgTime,
		AllocatedMB: message.AllocatedMB,
		AllocatedVCores: message.AllocatedVCores,
		AllocatedContainers: message.AllocatedContainers,
		AggregateContainersAllocated: message.AggregateContainersAllocated,
		AggregateContainersReleased: message.AggregateContainersReleased,
		AvailableMB: message.AvailableMB,
		AvailableVCores: message.AvailableVCores,
		PendingMB: message.PendingMB,
		PendingVCores: message.PendingVCores,
		PendingContainers: message.PendingContainers,
		NumberOfNodes: message.NumberOfNodes,
	}
}