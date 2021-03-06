// Copyright 2018 Delivery Hero Germany
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
//     Unless required by applicable law or agreed to in writing, software
//     distributed under the License is distributed on an "AS IS" BASIS,
//     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//     See the License for the specific language governing permissions and
//     limitations under the License.

package policies

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"obi/master/model"
	"obi/master/predictor"
	"obi/master/utils"
	"time"
)

// TimeoutScalingStep constant value by which scale at each timeout
const TimeoutScalingStep = 15
// TimeoutLength number of metric windows to receive before scaling
const TimeoutLength = 2
// TimeoutPolicyUpperBound maximum number of scaling factor
const TimeoutPolicyUpperBound = 50

// TimeoutPolicy this policy scales the cluster each time it receives
// a certain amount of activations
type TimeoutPolicy struct {
	scalingFactor int32
	record        *predictor.AutoscalerData
	count		  int
}

// NewTimeout creates a new timeout policy for autoscaler
func NewTimeout() *TimeoutPolicy {
	// For later random number generation
	rand.Seed(time.Now().UTC().UnixNano())

	return &TimeoutPolicy{
		0,
		nil,
		TimeoutLength,
	}
}

// Apply scale based on a timeout: if it expires, add a node
func (p *TimeoutPolicy) Apply(metricsWindow *utils.ConcurrentSlice) int32 {
	var previousMetrics model.HeartbeatMessage
	var throughput float32
	var pendingGrowthRate float32
	var count int8
	var performance float32

	logrus.Info("Applying timeout-based policy")
	for obj := range metricsWindow.Iter() {
		if obj.Value == nil {
			continue
		}

		hb := obj.Value.(model.HeartbeatMessage)

		if previousMetrics.ClusterName != "" {
			throughput += float32(hb.AggregateContainersReleased - previousMetrics.AggregateContainersReleased)
			if hb.PendingContainers > 0 {
				memoryContainer := hb.PendingMB / hb.PendingContainers
				containersWillConsumed := hb.AvailableMB / memoryContainer
				pendingGrowth := float32(hb.PendingContainers - containersWillConsumed - previousMetrics.PendingContainers)
				if pendingGrowth > 0 {
					pendingGrowthRate += pendingGrowth
				}
			}

			count++
		}
		previousMetrics = hb
	}

	if count > 0 {
		throughput /= float32(count)
		pendingGrowthRate /= float32(count)

		performance = throughput - pendingGrowthRate // I want to maximize this

		if p.record != nil {
			// If I have scaled, send data point
			p.record.MetricsAfter = &previousMetrics
			p.record.PerformanceAfter = performance
			// Send data point
			logrus.WithField("data", *p.record).Info("Sending autoscaler data to predictor")
			serverAddr := fmt.Sprintf("%s:%s",
				viper.GetString("predictorHost"),
				viper.GetString("predictorPort"))
			conn, err := grpc.Dial(serverAddr, grpc.WithInsecure()) // TODO: encrypt communication
			if err != nil {
				log.Fatalf("fail to dial: %v", err)
			}
			pClient := predictor.NewObiPredictorClient(conn)
			pClient.CollectAutoscalerData(context.Background(), p.record)
			// Clear data point
			p.record = nil
		}

		fmt.Printf("Throughput: %f\n", throughput)
		fmt.Printf("Pending rate: %f\n", pendingGrowthRate)

		// Scale up one at each time interval until we reach p threshold
		if p.count == 0 && previousMetrics.NumberOfNodes < TimeoutPolicyUpperBound {
			p.scalingFactor = rand.Int31n(TimeoutScalingStep - 1) + 1
			if rand.Float32() < 0.5 {
				p.scalingFactor *= -1
			}
			p.count = TimeoutLength
		} else {
			p.scalingFactor = 0
		}

		// Never scale below the admitted threshold
		if previousMetrics.NumberOfNodes + p.scalingFactor < LowerBoundNodes {
			p.scalingFactor = 0
		}

		p.count--
	}

	if p.scalingFactor != 0 && p.record == nil {
		// Before scaling, save metrics
		p.record = &predictor.AutoscalerData{
			Nodes:             previousMetrics.NumberOfNodes,
			PerformanceBefore: performance,
			ScalingFactor:     p.scalingFactor,
			MetricsBefore:     &previousMetrics,
		}
		logrus.WithField("data", p.record).Info("Created dataset record")
	}

	return p.scalingFactor
}
