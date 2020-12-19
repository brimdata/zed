# zqd recruiter

The zqd recruiter is started with `zqd --personality=recruiter`

zqd recruit manages a pool of zqd workers and provides a mechanism for zqd root processes to recruit zqd worker processes for query execution. It is designed to work well in a Kubernetes cluster, but it is not directly dependent on K8s APIs, and can function identically in other environments. This allows local development and ZTest scripts to be independent of K8s APIs.

## Design

The overall goal of the recruiter system is to provide workers for zqd root processes so distributed queries can be executed. We expect that query execution may be long-running (up to many minutes) and may be CPU and memory intensive. For these reasons we dedicate a worker to a single zqd root process for the duration of the distributed query. 

From the workers point of view, after it has been recruited by a zqd root process, it expects to receive search requests followed by a release request.

The recruiter has a /recruiter/register API by which the worker can register it's availablity to be recruited. The recruiter also has a /recruiter/recruit API for the zqd root process to recruit workers for distributed queries. The pool of workers is maintained in a Kubernetes cluster using a deployment that specifies the number of replicated workers. This replication count can be adjusted up or down based on autoscaling algorithms outside of the recruiter. Each worker in the cluster registers with the single recruiter in the cluster. An important design goal is that recruiter pod and worker pods can be restarted at any time with no noticable interuption of service.

## API

zqd personality=recruiter provides the following REST API:

### /recruiter/register

Request: {"addr" : "*host:port for worker*", "node" : "*ID of node in cluster*"}
Response: {"directive" : "reserved" OR "reregister"}

/register is called by zqd worker processes. It is a long-poll call: the connection will be held open by the recruiter until the worker is recruited or the call times out. /register is called in a goroutine loop.

When the recruiter process receives a /register request, it adds that worker to a pool of available workers. It maintains a list of available workers for each node in the cluster. Before responding the long-poll with "reserved" or "reregister", the recruiter removes the worker from the available pool. Thus, a worker can only be recruited when it has an active connection to the recruiter, i.e. we know it has not crashed.

When a worker receives a "reregister" response from the recruiter, it sends another /register request without delay.
If the worker get an error on registration, it retrys the request with an exponential back-off (delay).

When a worker receives a "reserved" response, it enters a reserved state in which it will expect to receive a series of `/worker/chunksearch` requests followed by a `/worker/release` request. In this state it has a timer: if it exceeds the timeout without receiving the requests, the process will exit.

### /recruiter/recruit

Request: {"N":*number of workers requested*}  
Response: {"workers":[ *list of recruited workers addr,node* ]}

`/recruit` is called by a zqd root process prior to starting query execution (i.e. `/search`). The number of workers returned may be less than requested, based on availability.

`/recruit` implements a heuristic algorithm to minimize the number of recruited worker instances from a single node. For example, if it is possible to pick each worker from a separate node, `/recruit` will do so. The goal of this heuristic is to schedule the worker on a node where it will not be directly competing with its peer workers for S3 bandwidth.

When a worker is recruited it is removed from the available pool, the worker enters a state where it knows it has been reserved. It will not reregister until after the root process sends it a `/worker/release` message.

## Recovery on worker restart

When a worker halts for any reason, while it is registered with the recruiter, the recruiter will lose the connection and the worker will be removed from the available list.

When a worker restarts, it will attempt to register with a recruiter.

The recruiter does not keep any state for previously registered workers.

If a worker halts while working on a distributed query, that worker is lost to the zqd root process. (In the future, we can add logic to allow a zqd root process to replace a lost worker.)

## Recovery on recruiter restart

In a cluster environment, it should not be surprising that any one pod will restart occasionally to be reschduled on another node. So, any clustered application should not depend on continious memory state of one pod. Of course there are many other potential reasons for an unexpected shutdown of a pod.

The current design calls for one recruiter per cluster. When that recruiter unexpectly restarts, all registered worker pods will be in a loop that will cause them to reregister after a short delay specified in their config (e.g. 200 ms with an exponential backoff). In a healthy cluster, restarting a recruiter will be sub-second, and all available workers will reregister shortly thereafter. No information is lost, and the recruiter is available for `/recruit` requests without a significant interuption.

The recruiter is "more or less" a stateless service, given that it only persists state about its current open connections. So, if we want to, we could safely run more than one instance of a recruiter in the same cluster. Statistically, we would expect the two instances to split the worker pool into two similarly sized partitions with no overlap. This would only be favorable for large clusters where a smaller pool would not lead to suboptimal scheduling. It might be a good idea for clusters that have more than a few hundred available workers. In any case, the ablity to run two recruiters without conflict could be helpful for a zero-downtime rolling upgrade.

## Recovery on query root process restart

On a `/recruit` request, the recruiter process responds to each worker, informing the worker that it is in a "reserved" state. The recruiter then responds to the zqd root process with a list of the URLs of the recruited workers. (At that point the recruiter forgets about the workers.) In the "reserved" state, each worker knows it should be receiving work, and waits for requests from a query root process.

If the root process halts (restarts) before sending requests, the workers have a specified "idle" timeout, after which they will exit. The exit will cause the worker pods to be restarted (and possibly scheduled to different nodes) and they will register with the recuiter as they come up.

If the root process halts (crashes for any reason) during query execution, the connection from the root to the worker will be broken. This will cause the idle timer in that worker to be restarted, and the worker will exit unless the root sends it another reuqest before the timeout.

Suppose the root process gets into "wedged" state where it keeps the connection with the workers open, but is not making progress toward completing a query. In that case, the "wedged" root process will also cause the workers to be unavailable for recruitment. It may be worth adding code to the zqd query path to detect this type of failure.

Under normal circumstances the zqd root process will send a `/worker/release` request that allows the worker to gracefully exit the "reserved" state. If that does not happen, and the zqd root is not holding open a connection with a `/worker/chunksearch` request, then the workers will timeout and exit.

## Some thoughts for future K8s integration

The recruiter may be able to contribute to autoscaling heuristics. It could publish metrics that are used by the Horizontal Pod Autoscaler (HPA) to scale the number of workers in a cluster up and down.

In addition, the workers could read details of their K8s pod prior to sending a register request. If the pod's node is marked as unschedulable by K8s, then the worker could decline to register.
