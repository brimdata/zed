# zqd recruiter

The zqd recruiter is started with `zqd --personality=recruiter`

zqd recruit manages a pool of zqd workers and provides a mechanism for zqd root processes to recruit zqd worker processes for query execution. It is designed to work well in a Kubernetes cluster, but it is not directly dependent on K8s APIs, and can function identically in other environments. This allows local development and ZTest scripts to be independent of K8s APIs.

When zqd workers and root processes are started with the zqd listen command, they may be provided with the endpoint of the zqd recruit process.

## API

zqd recruit provides the following REST API:

### /register

{  
"addr" : "*host:port for worker*",  
"node" : "*ID of node in cluster*"  
}

/register is called by a worker process after it has started and is capable of processing /worker messages.

/register will be called again when a zqd process completes a /worker request and is available to be recruited by another zqd root process for query execution. 

When the recruiter process receives a /register request, it adds that worker to a pool of available workers. It maintains a list of available workers for each node in the cluster.

In order to make the recruiter process state recoverable, /register will be called periodically when a  worker process is idle (e.g. not processing /worker requests) for a given timeout period. 
This will allow a restarted zqd recruiter process to get registrations from running zqd workers.

/register is a noop if the address is already registered, or if the worker is in the “reserved” pool. (See /unreserve below.) 

### /recruit

Request: {"N":*number of workers requested*}  
Response: {"workers":[ *list of recruited workers addr,node* ]}

/recruit is called by a zqd root process prior to starting query execution (i.e. /search). The number of workers returned may be less than requested, based on availability.

/recruit implements a heuristic algorithm to minimize the number of recruited worker instances from a single node. For example, if it is possible to pick each worker from a separate node, /recruit will do so.

When a worker is recruited, the worker is added to the “reserved pool” so it cannot be inadvertently re-registered and recruited again until it has been unreserved. See the /unreserve API below.

### /unregister

{"addr":"*host:port for worker*"}

/unregister is called by a zqd worker process that gracefully terminates while “believing” that it is registered with the zqd recruiter.

### /unreserve

{"addrs":["*host:port for worker*",...]}

/unreserve is called by a zqd worker process that becomes idle after having completed a /worker request. It is removed from the reserved pool. This is a noop if it was not in the reserved pool.

/unreserve is also called by a zqd worker process when it starts up, in case the addr of the worker was reserved by a previous process that is no longer running.

The /unreserve API, along with the reserved pool maintained by /recruit, avoids the potential race condition of a worker re-registering itself after it has been recruited but before it has been sent work by a zqd root process.

## Analysis of failures

1. If the zqd root process halts after receiving a reply to the /recruit message and sending /worker messages to the recruited workers, then the recruited workers will be unavailable until they have an idle time out and send /register again. This could be a negative impact if the idle timeout for /register is long.

2. There is a window of time between the recruiter process replying to a /recruit and the worker process becoming busy in which the worker could resend a /register message. In this case, the initial /recruit transaction will have already added the worker to the reserved pool, the the errant /register request will be ignored. The worker (or some other process) must call /unreserve before the worker will be reregistered.

3. If a worker process halts without sending /unregister, then it can be recruited with a call to /recruit. In this case, the recruiting agent (e.g. the zqd root) must be tolerant of the fault, and detect that it cannot reach the worker. It should either do without, or send another /recruit request to get a replacement.

4. Each of the API calls (/register /unregister /unreserve /recruit) may only mutate the state of the freePool, nodePool, and reservedPool as an atomic transaction. In this initial implementation, this is easy to guarantee because all the data is in-memory. The four transactions "can't fail" -- that is, any failure is a panic that will halt the process. (If we move to an exernal database implementation, we would need to build a transaction manager to handle failure cases.) Out of memory failure is not likely, since the data structures are small. I think deadlock is practically impossible since we share one mutex for all three maps and all four operations, but pending transactions could get backed if they are not completed as quickly as they are received. For very large node pools (> 1000 nodes) the current implementation will be too slow because the keys for the nodePool are shuffled as part of the recruit. (As a practical matter, it is unlikely we will want to deploy K8s clusters with >1000 nodes.)

## Database

The initial implementation of zqd recruit will not require an external database. The use of the /register request by zqd workers will ensure that the available pool is eventually consistent with the state of the zqd workers. This allows the zqd worker instances to manage their own lifecycle, and terminate and restart when appropriate. In a K8s cluster, high availability of the zqd recruit process will be based on its ability to rapidly restart and recover state from incoming /register requests.

Future implementations may use an external Redis database. This will have the advantage of allowing a zqd recruiter instance to more quickly recover all state after an unexpected restart. It would also allow us to run more than one instance of a zqd recruiter process per cluster. The CPU requirements and memory requirements for a zqd recruiter process are likely to be small, but we may find that having extra instances improves availability.

The main reason to introduce Redis will come when we introduce SSD caches for nodes running zqd workers. We can then use cache state (e.g. the availability of desired S3 objects) to preferentially schedule zqd workers on a given node. This heuristic for scheduling will require up-to-date information on the cache state of each node in the cluster, and a Redis database is a convenient way to share information on recent cache state.

## Future plans for autoscaling

A future implementation of the zqd recruiter process will monitor the rate of /recruit request and the size of the available pool. If the pool is smaller or larger than desired for a given period of time, the zqd recruiter could trigger a scaling process for the zqd worker instances. For example, it could adjust the number of replicas in a K8s deployment. This type of autoscaling is more targeted and precise for our needs than the behavior of the K8s Horizontal Pod Autoscaler (HPA).

## Future topic: "Cache striping"

When we implement SSD caches for K8s cluster nodes that host zqd workers, we will want to implement scheduling heuristics that make it more likely that S3 objects are cached in such a way that searches will be executed in parallel. As an example, suppose that we are performing a search that engages W worker processes. As mentioned above, we would like the workers to run on different nodes, to spread the CPU load. In addition, when workers have a "cache miss" and must perform an S3 GET, we would like the retrieved S3 objects to be "spread evenly" across the available nodes. It would be optimal for S3 objects corresponding to consecutive time spans of data to be cached on different nodes. Ideally, the cached S3 object for a long time span, say N consecutive S3 objects, would be cached on N separate nodes. This is analogous to "striping" data on RAID arrays. The scheduling heuristics for the zqd recruiter process will influence how the SSD cache can be best utilized.
