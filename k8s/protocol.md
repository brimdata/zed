*This is work in progress! (pre-alpha :-)*

# Protocol for distributed zqd

zqd executes queries by decomposing the user provided ZQL into an abstract syntaxt tree (AST) of "proc" objects. The procs are the nodes in the AST, and are executee the "atoms" of query processing -- a simple example of a proc would be a string match condition on a single column. (There is more about procs in other docs.) 

Note: although we call this an AST in the code, I think it is a directed acyclic graph (DAG) that begins with a single node and ends with a single node. In the following dicussion I make that assumption.

The basic strategy of distributed zqd is to allow procs to be executed by remote processes. This describes, at a high level, the protocol between a  "manager" zqd process, that builds the AST of procs, and "worker" zqd processes that execute a subtree of the manager's AST.

## Code Story

Here I will assume the the "manager" and the "worker" zqd have different features -- they are not necessaily interchangable.

We start by walking through a the logic of query processing. 

1. The manager zqd process receives a query though its REST API, e.g. Brim sends it a query.
2. The manager parses the query and creates an AST of procs.
3. We run an algorithm on the AST to detect procs that may be executed in parallel. (TODO: need a code reference here. This is currently implemeted to allow for multi-threading, but not yet for multi-processing. The algorithm currently "clones" the procs that may be run in parallel, transforming the DAG into a new DAG with more nodes.)
4. When when a proc may be run in parallel, the manager process uses a heuristic to determine how many "worker" processes to recruit to do the work. Note that we say "recruit" rather than spawn, because the manager does not spawn the worker processes directly. We will use K8s to create workers within the same cluster as the manager. The heuristic for how many worker to recruit could depend on (a) the number of files or S3 objects that are sources of data, (b) the number or records in the files, (c) user preferences, such as a limit for the maximum number of worker to recruit (more later!)
5. When the manager recruits a worker, it obtains a TCP endpoint with which to communicate with the worker. (This endpoint resolves though the K8s DNS, so it looks like host:port.)
6. At a low-level, we assume the protocol between the manager and worker is bidirectional and stream-like (i.e. not a REST API). The protocol will include ZNG over TCP. We will define our own frame, most likely an integer protocol version, followed by an integer frame length, followed by binary ZNG. (The advantage of using ZNG is that we will will not have to transform the message content prior to working on it in zqd.)
7. In addition to the sequence of types and records in ZNG, there is additional information that the manager and worker processes must exchange: the manager must initiate the conversation with the worker by sending a serialization of an AST of procs that represents the subset of the manager's AST that is being delegated to the worker. The serialized proc must contain a reference to the source of data which the worker will process -- we will call this combination AST + datasource. Simple examples of data sources are files or S3 objects. More complex data sources can be other workers that have been recruited by the same manager, to work on other parts of the manager's AST (more on worker-to-worker streams later.)
8. The protocol is bidirectional. As the worker completes "work" the results (in ZNG form) are streamed back to the manager on the same TCP connection. It is up to the manager to determine what to do with the returned "work". Typically is will be the manager's reponsiblity to merge the streams of completed work from multiple worker into one stream that will be returned from the manager to the original caller, i.e. Brim. Another option that the manager has is to return a "redirect" response to the worker. (More on redirects later.)
9. After the worker has completed the work specified in the AST + datasource initially sent by the manager, it will send a 'DONE' message to notify the manager that it needs more work to do. At this point the manager can send it another AST + data source "job" to do. When the manager replies to the worker's 'DONE' with it's own 'DONE' then the worker will terminate. The manager will send a DONE reply to the worker when it does not have another "job" for the worker -- then the worker process will terminate gracefully.
10. Query processing completes when the AST completes for the manager process - just like it does currently for zqd. By the time query processing completes, all worker processes recruited by the manager will have sent a 'DONE', received a 'DONE' and terminated.

We do not yet discuss failure cases, like workers stranded with no manager. We will need some features like worker time-out and manager re-tries to make this resilient.

Also note that multi-threading is not mentioned. I think we should get the multi-porcessing pattern working before thinking about how it plays with multi-threading, or whether we need both patterns in the same compute cluster. (In any case, we probably want to optimize for best bandwidth to S3 in preference to optimal use of CPU for context switching.)

## How this works for group-by procs (Code Story 2)

Here is how the protocol handles high-cardinality group-by operations, in a pattern similar to the zar continuous shuffle discussion.

This is where we get back to a discussion of worker-to-worker streams and redirect responses.

Let's say the AST of procs contains a filter proc followed by a group-by on a key of unknown cardinality.

Early in the process, we know the cardinality of rows to be filtered, and the number of files or S3 objects that contain rows. This allows the manager process to recruit an appropriate number of workers to do the filtering in parallel. This follows the pattern above (in 1-10).

1. Initially, the manager process is assumed to be handling the group-by. The workers will stream their completed work (on filtering) back to the manager. 
2. At some point, the manager may determine the the cardinality of the group-by keyspace has grown too large to be efficiently handled by a single process. (If that never happens, then stop here.)
3. At that point, the manager will recruit two (or more) workers to process the group-by. The group-by key space is split based on a hash function (e.g. xxhash or some other uniformly distributed non-cryptographic hash function.) 
4. The manager splits its partially completed work on the group-by, based on the hash key, and sends "half" of it to the first recruit and the other "half" to the second recruit.
5. The manager will continue to receive completed filtered records from the workers recruited to complete the filtering work. However, it no longer wants to do the group-by work itself. Instead, it wants to delegate it to the group-by workers it recruited. So, on the TCP connections it uses to receive work from the filter workers, it sends a "REDIRECT" message that includes a table of the workers performing the group-by work, and a hashing rule so the filter workers can determine which group-by worker should receive their completed work. (Call this the "routing table".)
6. The REDIRECT message is sent asynchonously, on bidirectional connection with the worker. Since there is no synchronous response, the manager will forward completed work it receives from filter workers in the meantime. (If some time passes and it looks like the REDIRECT has not been processed, the manager should retry sending it. BTW, in general, this whole protocol is only going to work well on reliable high-bandwidth connections, e.g. in-cluster for K8s.)
7. At some point, a worker may realize it is running out of key space, and make a decision to split its work, in the same way as the manager did previously. It will then send a "SPLIT" message to the manager, which will recruit another worker. When the new worker is available, the manager will send the new "routing table" to the worker that requested the SPLIT. That worker will forward "half" of its existing group-by keys to the new worker. It will start to buffer records that should be processed by the new worker until the new worker has received the existing keys and has "come online".
8. When the new worker is ready (has received the work-in-pogress group-by keys) it notifies the manager. The manager then broadcasts a new "routing table" to all the filter workers.
9. The filter workers are now sending their work directly to the group-by workers. When they are have completed work they send a "DONE" response to the manager. When all filter workers have sent a "DONE" response to the manager, it broadcasts a "FINISH" message to all of the group-by workers.
10. When a group-by worker receives a "FINISH" from the manager, it starts to sort the group-by keys (if they are not already in sorted order) and prepare them to be streamed to the manager.
11. The manager reads from the streams of all the group-by workers, in sorted order, so it is able to merge the group-by results from the N worker streams into a single result stream to return to the original caller, e.g. Brim.

I know a lot of this is just a re-wording of how the group-by code already works. I'm just restating it here to put it into the context of bidirectional asyncronous interprocess communication.

When I wrote the heading "Code Story" I was thinking of the green plastic "army men" -- so maybe we should call the processes "sergeants and specialists" instead of "managers and workers".
