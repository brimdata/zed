# ZNG over HTTP

ZNG over HTTP is a very simple transport protocol to post a ZNG stream
to an http service.

The client sends data to a server with an http POST method
followed by streaming ZNG data in the body as chunked transfer encoding.
A client may persist the connection and stream live ZNG data as it becomes
available.

When a client finishes transmitting data, it sends a zero-length chunk and the
server responds with an http status OK if all data has been successfully received,
processed, and written to stable storage.  Otherwise, the server responds
with an http error status and a JSON failure message in the body of the response.

## Reliability

A simple at-least-once reliability semantic is achieved using a client-driven
protocol embedded as a ZNG control code in the ZNG data stream.

An HTTP-over-ZNG server must implement this reliability.
A client may or may not choose to use it.

To request acknowledgement of data received, the
client embeds a synchronization marker with ZNG control code 6
and a string marker to be transmitted back to the sender:
```
#!6:<marker>
```
where `<marker>` is an arbitrary string chosen by the client.  The server responds
to this message by streaming in the http response body an acknowledgement
of the form:
```
#!7:<marker>
```
This guarantees to the client that the server has received and processed
all of the transferred ZNG data without error up to the indicated marker.

For a stronger guarantee, the client may embed a flush directive:
```
#!8:<marker>
```
The server responds to a flush
by transmitting as the response body an acknowledgement of the form:
```
#!9:<marker>
```
This guarantees to the client that the server has received and processed
all of the transferred ZNG data without error up to the indicated marker and
committed the data or operation implied by the ZNG stream to durable storage.

Note that if a client never sends these control messages, then the server never
generates responses and simply operates normally.

## Backward compatibility

Since a Zeek log is fully interchangeable with the ZNG format, Zeek files can be
easily posted to a service like this using, e.g.,
```
zq -f zng cong.log > conn.zng
curl -X POST "http://localhost:8080/logs" --data-binary @conn.zng
```

## Reference Implementations

* Production client - [zeek-tsv-http-plugin](https://github.com/brimsec/zeek-tsv-http-plugin)
* Simple receiver/gateway server - [zinger](https://github.com/brimsec/zinger)
