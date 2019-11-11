# zson over http

zson over http is a very simple transport protocol to post zson data
to an http service.

The client sends data to a server with an http POST method
followed by streaming zson data in the body as chunked transfer encoding.
A client may persist the connection and stream live zson data as it becomes
available.

When a client finishes transmitting data, it sends a zero-length chunk and the
server responds with an http status OK if all data has been successfully received,
processed, and written to stable storage.  Otherwise, the server responds
with an http error status and a JSON failure message in the body of the response.

## Reliability

A simple at-least-once reliability semantic is achieved using a client-driven
protocol embedded as zson comments in the zson data stream.

An http-over-zson server must implement this reliability.
A client may or may not choose to use it.

To request acknowledgement of data received, the
client inserts a syn marker as a comment:
```
#!syn <id>
```
where \<id> is an arbitrary string chosen by the client.  The server responds
to a syn by streaming in the http response body an acknowledgement of the form:
```
#!ack <id>
```
This guarantees to the client that the server has received and processed
all of the transferred zson data without error up to the indicated syn.

For a stronger guarantee, the client may send a flush directive:
```
#!flush <id>
```
The server responds to a flush
by transmitting as the response body an acknowledgement of the form:
```
#!ack <id>
```
This guarantees to the client that the server has received and processed
all of the transferred zson data without error up to the indicated syn and
committed the data or operation implied by the zson to durable storage.

Note that if a client never sends syn/flush directives, then the server never
generates responses and simply operates normally.

## Backward compatibility

Since a zeek file is by definition a zson file, zeek files can be posted to
a service like this using, e.g.,
```
curl -X POST "http://localhost:8080/logs" --data-binary @conn.log
```

## Reference Implementations

* production client - [zson-http-plugin](https://github.com/looky-cloud/zson-http-plugin)
* toy server - [zsond](https://github.com/mccanne/zsond)
