# Copyright 2018 Delivery Hero Germany
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
#     Unless required by applicable law or agreed to in writing, software
#     distributed under the License is distributed on an "AS IS" BASIS,
#     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#     See the License for the specific language governing permissions and
#     limitations under the License.

import json
import os
import socket
import sys
import urllib.request

HOSTNAME = socket.gethostname()
CLUSTER_NAME = HOSTNAME[:-2]

# Before doing anything, make sure that the current node is the master-old
GET_MASTER_CMD = '/usr/share/google/get_metadata_value ' \
                 'attributes/dataproc-master '
master_name = os.popen(GET_MASTER_CMD).read()
if master_name != HOSTNAME:
    # If we are not in the master-old we should not send any heartbeat
    # so the current program can be aborted
    sys.exit(1)


# Get metadata information
def get_metadata(attr):
    return os.popen('/usr/share/google/'
                    'get_metadata_value attributes/' +
                    attr).read()


QUERY = 'jmx?qry=Hadoop:service=ResourceManager,name=QueueMetrics,q0=root,' \
        'q1=default'
QUERY_URL = 'http://{}:8088/{}'.format(HOSTNAME, QUERY)

RECEIVER_ADDRESS = get_metadata('obi-hb-host')

RECEIVER_PORT = get_metadata('obi-hb-port')
RECEIVER_PORT = int(RECEIVER_PORT)

NORMAL_NODE_COST = get_metadata('normal-node-cost')
NORMAL_NODE_COST = float(NORMAL_NODE_COST)
PREEMPTIBLE_NODE_COST = get_metadata('preemptible-node-cost')
PREEMPTIBLE_NODE_COST = float(PREEMPTIBLE_NODE_COST)
NODE_DISK_SIZE = get_metadata('node-disk-size')
NODE_DISK_SIZE = int(NODE_DISK_SIZE)
DISK_COST = get_metadata('disk-cost')
DISK_COST = float(DISK_COST)
DATAPROC_NODE_COST = get_metadata('dp-node-cost')
DATAPROC_NODE_COST = float(DATAPROC_NODE_COST)
INTERVAL = get_metadata('interval')
INTERVAL = int(INTERVAL)

# Get current cumulative cost
cost_path = os.path.join('/tmp', 'cumulative_dataproc_cost')
cumulative_cost = 0.0
if os.path.exists(cost_path):
    with open(cost_path, 'r') as f:
        try:
            cumulative_cost = float(f.read())
        except ValueError:
            cumulative_cost = 0.0


def get_nodes_count():
    normal = os.popen(
        "yarn node -list 2> /dev/null | "
        "egrep '^obi-[^-]+-w' | wc -l").read()
    preemptible = os.popen(
        "yarn node -list 2> /dev/null | "
        "egrep '^obi-[^-]+-sw' | wc -l").read()
    return normal, preemptible


def send_hb():
    # Get Heartbeat message and serialized it
    hb = compute_hb()
    serialized = hb.SerializeToString()

    # Create UDP socket object for sending heartbeat
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

    # Send heartbeat through UDP connection
    sock.sendto(serialized, (RECEIVER_ADDRESS, RECEIVER_PORT))

    # Close socket connection
    sock.close()


def compute_hb():
    # Initialize HB message
    hb = HeartbeatMessage()
    hb.cluster_name = CLUSTER_NAME

    # Collect metrics from Yarn master-old
    req = urllib.request.urlopen(QUERY_URL)
    req = req.read().decode('utf-8')
    metrics = json.loads(req)
    metrics = metrics['beans'][0]

    # Finish building heartbeat message
    hb.AMResourceLimitMB = metrics['AMResourceLimitMB']
    hb.AMResourceLimitVCores = metrics['AMResourceLimitVCores']
    hb.UsedAMResourceMB = metrics['UsedAMResourceMB']
    hb.UsedAMResourceVCores = metrics['UsedAMResourceVCores']
    hb.AppsSubmitted = metrics['AppsSubmitted']
    hb.AppsRunning = metrics['AppsRunning']
    hb.AppsPending = metrics['AppsPending']
    hb.AppsCompleted = metrics['AppsCompleted']
    hb.AppsKilled = metrics['AppsKilled']
    hb.AppsFailed = metrics['AppsFailed']
    hb.AggregateContainersPreempted = metrics['AggregateContainersPreempted']
    hb.ActiveApplications = metrics['ActiveApplications']
    hb.AppAttemptFirstContainerAllocationDelayNumOps = \
        metrics['AppAttemptFirstContainerAllocationDelayNumOps']
    hb.AppAttemptFirstContainerAllocationDelayAvgTime = \
        metrics['AppAttemptFirstContainerAllocationDelayAvgTime']
    hb.AllocatedMB = metrics['AllocatedMB']
    hb.AllocatedVCores = metrics['AllocatedVCores']
    hb.AllocatedContainers = metrics['AllocatedContainers']
    hb.AggregateContainersAllocated = metrics['AggregateContainersAllocated']
    hb.AggregateContainersReleased = metrics['AggregateContainersReleased']
    hb.AvailableMB = metrics['AvailableMB']
    hb.AvailableVCores = metrics['AvailableVCores']
    hb.PendingMB = metrics['PendingMB']
    hb.PendingVCores = metrics['PendingVCores']
    hb.PendingContainers = metrics['PendingContainers']

    # Collect number of nodes
    n_nodes = os.popen("yarn node -list 2> /dev/null | grep 'Total Nodes:' "
                       "| egrep -o '[0-9]+'").read()
    hb.NumberOfNodes = int(n_nodes)

    # Timestamp
    hb.Timestamp.GetCurrentTime()

    # Compute new cumulative cost and store it
    global cumulative_cost

    normal_nodes = int(os.popen("yarn node -list 2>/dev/null "
                            "| egrep '[a-zA-Z0-9]+\-[a-zA-Z0-9]+\-w\-' "
                            "| wc -l").read()) + 1
    secondary_nodes = int(os.popen("yarn node -list 2>/dev/null "
                                   "| egrep '[a-zA-Z0-9]+\-[a-zA-Z0-9]+\-sw\-' "
                                   "| wc -l").read())

    gce_cost = normal_nodes * NORMAL_NODE_COST + secondary_nodes * PREEMPTIBLE_NODE_COST

    disk_cost = (normal_nodes + secondary_nodes) * NODE_DISK_SIZE * DISK_COST

    dataproc_cost = DATAPROC_NODE_COST * (normal_nodes + secondary_nodes)

    current_cost = INTERVAL * (
            gce_cost +
            disk_cost +
            dataproc_cost
    )
    cumulative_cost += current_cost

    with open(cost_path, 'w+') as f:
        f.write(str(cumulative_cost))

    hb.Cost = cumulative_cost

    return hb


if __name__ == '__main__':
    send_hb()
