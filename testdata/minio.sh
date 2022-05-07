#!/bin/bash

mkdir -p data/bucket

# Allocate a port.  Another process could bind to it before MinIO does,
# but that's very unlikely.
port=$(python3 -c "import socket; print(socket.create_server(('localhost', 0)).getsockname()[1])")
minio server --address localhost:$port --console-address localhost:0 --quiet data &
trap "kill -9 $!" EXIT

# Wait for MinIO to accept a connection.
python3 <<EOF
import socket, time
start = time.time()
while True:
    try:
        socket.create_connection(('localhost', $port))
        break
    except ConnectionRefusedError:
        if time.time() - start > 5:
            raise
    time.sleep(0.1)
EOF

export AWS_REGION=does-not-matter
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:$port
