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

package heartbeat

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"net"
			"obi/master/model"
		"obi/master/pool"
	"obi/master/persistent"
	"obi/master/platforms"
	"obi/master/autoscaler/policies"
	"obi/master/autoscaler"
)

// Receiver is the heartbeat module in charge of updating clusters metrics.
// In a long-living routing it listens for all the incoming heartbeats from cluster masters.
// If it receives an heartbeat from a cluster not in the pool, it creates the instance
// for that cluster in order to monitor it.
type Receiver struct {
}

// channel to interrupt the heartbeat receiver routine
var quit chan struct{}

// UDP connection
var conn *net.UDPConn

// New is the constructor of the heartbeat Receiver struct
// @param pool contains the clusters to update regularly
// return the pointer to the instance
func New() *Receiver {
	r := &Receiver{}

	return r
}

// Start the execution of the heartbeat receiver
func (receiver *Receiver) Start() {
	quit = make(chan struct{})
	logrus.Info("Starting heartbeat receiver routine.")
	go receiverRoutine(pool.GetPool())
}

// goroutine which listens to new heartbeats from cluster masters. It will be stop when an empty object is inserted in
// the `quit` channel
// @param pool contains the available clusters to update with new metrics
func receiverRoutine(pool *pool.Pool) {
	var err error

	// listen to incoming udp packets
	addr := net.UDPAddr{
		Port: 8080,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		logrus.WithField("error", err).Error("'ListenUDP' method call for creating new UDP server failed")
		return
	}

	for {
		data := make([]byte, 4096)
		n, err:= conn.Read(data)
		if err != nil {
			select {
			case <-quit:
				logrus.Info("Closing heartbeat receiver routine.")
				// the error was caused by the closing of the listener
				return
			default:
				// temporary error - let's continue
				continue
			}

		}

		m := model.HeartbeatMessage{}
		err = proto.Unmarshal(data[0:n], &m)

		if err != nil {
			logrus.WithField("error", err).Error("'Unmarshal' method call for new heartbeat message failed")
			continue
		}

		if value, ok := pool.GetCluster(m.GetClusterName()); ok {
			cluster := value.(model.ClusterBaseInterface)
			cluster.AddMetricsSnapshot(m)
			logrus.WithField("clusterName", m.GetClusterName()).Info("Metrics updated")
		} else {
			logrus.WithField("clusterName", m.GetClusterName()).Info("Received metrics for a cluster not in the pool.")

			clusterExists, err := persistent.ClusterExists(m.GetClusterName())
			if err != nil {
				continue
			}

			newCluster, err := platforms.NewExistingCluster("dataproc", m.GetClusterName())
			if err != nil {
				logrus.WithField("Error", err).Error("Existing cluster not inserted in the pool")
				continue

			}

			if clusterExists {
				policy :=  policies.NewWorkload(0.2)
				a := autoscaler.New(policy, 60, newCluster.(model.Scalable), false, 0)

				pool.AddCluster(newCluster, a)

				a.StartMonitoring()

				logrus.WithField("clusterName", m.GetClusterName()).Info("Added cluster in the pool")
			} else {
				newCluster.FreeResources()
			}
		}
	}
}

// Stop the execution of the receiver goroutines
func (receiver *Receiver) Stop() {
	close(quit)
	conn.Close()
}
