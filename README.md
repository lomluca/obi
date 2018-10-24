# OBI (Objectively Better Infrastructure)

OBI is a project from Delivery Hero's Data Insight team which represents an
attempt to optimize clusters resource utilization in order to limit their
operational costs. 

Project goals:

 - Optimize resource usage for low dimensional topologies (e.g., Borg/k8s was
   developed for contexts with vast amount of resources).
 - Delegate the entire value chain of data analysis to the end user removing the
   need for Platform Ops support.
 - Support automation of Data Operations: a user who wants run data analytics
   applications should not be bothered by system details, such as how to
   configure the amount of RAM a Spark Executor should use, how many cores are
   available in the system or even how many worker nodes should be used to meet
   an execution deadline.

## Code Structure

 - `api` web server exposing OBI's internal database for external usage e.g.
   querying the status of a job while it's running
 - `assets` generic assets e.g. images used in the code
 - `client` contains the code for using OBI from a remote client. Two clients
   are made available: the first one, written in Python and accessible through
   the `client/generic_client.py` script is meant for system administrators and
   another one, accessible by compiling `client/obi-submit.go` which is meant
   for the final user to allow him to submit his jobs
 - `examples` contains example YAML files to showcase how a system administrator
   can attach a cloud computing platform with an OBI deployment
 - `master/autoscaler` code written for the autoscaler feature
 - `master/heartbeat` set of scripts and `protobuf` schemas used to run the
   heartbeat service on clusters master node and to collect them into OBI
   architecture
 - `master/model` definition of generic data structure, which define how each
   cluster should be interfaced
 - `master/persistent` code written to allow OBI to connect to a database for
   persisting its state
 - `master/platforms` implementations of the generic interfaces from
   `master/model`, capable of extending OBI to each possible cloud computing
   service
 - `master/pool` code allowing the handling of multiple cluster along with
   utility functions to allocate them
 - `master/predictor` contains code which is autogenerated to allow
   communication between OBI Master and the predictor component
 - `master/scheduling` contains the logic for the OBI scheduler
 - `master/utils` general utility functions
 - `predictor` web server listening for requests to the predictive component
 - `predictor/predictors` machine learning models used to provide predictions
   different from job duration
 - `proto` Google Proto Buffer files used to generate RPC communication
   interfaces between components

## Architecture

OBI's architecture was designed to be a microservice architecture. In the below
figure, each block but "DATAPROC" are supposed to be separated microservices,
communicating between each other in different ways, depending on the needs. For
example, given the requirement of high performance communication between the
clusters master node and the heartbeat component, those communication is handled
through Google's `protobuf` schemas. All the communication between components
happens trhough RPC while the API component can be accessed through HTTP
requests.

![alt text](assets/obi-architecture.jpg "OBI Architecture")

### OBI Workflow

OBI is meant to provide an abstraction layer between the user and the cluster
infrastructure, in order to simplify the job submission process and to optimize
operational costs.

OBI jobs may have different priority levels. If a job having maximum priority is
submitted, OBI will create a cluster which will be exclusivelly allocated to
that job. Otherwise, if a job with lower priority is submitted, OBI will try to
pack it with similar jobs in bins, following a certain policy.

OBI scheduler supports two scheduling policies while packaging jobs into bins:
 - **count based**: bin are filled with jobs coming from the same priority band
   up to a certain count
 - **time based**: in this policy the scheduler asks the predictor module to
   generate an estimation of how long a certain job will last and then tries to
   pack jobs with the same priority into homogenous bins e.g. each bin should
   contain jobs for a maximum total duration of 1 hour

The scheduling policy and the priority levels are configurable through the OBI
deployment YAML file.

After jobs are scheduled and one or more clusters are allocated, each cluster is
assigned with an autoscaler routine. This routine will monitor the cluster
status through the heartbeat the OBI Master receives from them. Those heartbeats
contain information about the resources utilization in the cluster which are
used by the autoscaler to decide whether to add (or remove) nodes.

The goal of the autoscaler is to make the cluster working with the least
possible amount of nodes at each time step. Indeed, each cluster created by OBI
starts with the minimum possible amount of nodes, which are dynamically adjusted
according to the needs of the running jobs.

## Development

The first step to compile OBI is to generate the `protobuf` code required for
RPC communication between the components. Both
[protobuf](https://developers.google.com/protocol-buffers/) and
[gRPC](https://grpc.io) are strict requirements for this process. Keep in mind
that OBI is written mainly in Go with some parts in Python so the dependencies
have to be satisfied for both languages.

To generate the `protobuf` code we provide a script, `proto-gen.sh` which will
take care of all the development process.

After the `protobuf` files are generated, for each component you can find a
`Dockerfile` which can be used to create images from them to be deployed
anywhere.

We have also provided a client which can automatically deploy OBI on your
platform. So far, only Kubernetes is supported as a container orchestration
deployment platform. In order to deploy OBI in your Kubernetes infrastructure
you can use the `master/generic_client.py` script which is supposed to be a
client for system adiministrators.

Assuming that you put your OBI deployment YAML file in
`examples/deployment.yaml`, you can deploy a new OBI with the following command:

```bash
python3 client/generic_client.py  create infrastructure -f examples/deployment.yaml
```

Once an OBI instance is deployed, jobs can be submitted to it. In order to do
that you need to compile the OBI submitter client:

```bash
$ cd client/
$ go get .
$ got build .
```

This process requires you to install Go lang compiler and will produce you a
`client` executable as output. Jobs can be submitted using the following
command:

```bash
$ ./client -f JOB_SCRIPT_PATH -t PySpark -i OBI_DEPLOYMENT_NAME -p PRIORITY -- EXE_ARGS
```

The executable path be either a Google Cloud Storage URI or local executable.
The `-t` option defines the type of the job the user is submitting and, so far,
only PySpark is supported. The `OBI_DEPLOYMENT_NAME` is the one defined in the
OBI deployment YAML file when the OBI instance was created.

When submitting a job, the user is asked for username and password. Those values
should be written by the system administrator  in the `Users` table of the SQL 
database which is used by OBI to store its state.

### Helm Chart

In order to make the Kubernetes deployment process easier, we have also provided
an Helm chart.

Let's download the helm chart with:

```bash
$ git clone https://gitlab.dataops.deliveryhero.de/obi/helm-chart
```
Open `values.yaml` and fill in all the empty fields. Once the configuration is
completed, just deploy on your Kubernetes cluster with"

```bash
$ helm install obi-chart
```

## Contributions

### Integrate OBI in your infrastructure

OBI supports only Dataproc as a cloud computing service for now. However, it is
easily extensible to support new platforms by simply adding new code for it. In
fact, you would need to just add the code for you platform under the
`master/platforms` folder. In this code you should provide your own struct
extending `model.ClusterBase` and implementing `model.ClusterBaseInterface`.

As a refence, you can have a look at the existing Dataproc implementation
available in `master/platforms/dataproc.go`.
