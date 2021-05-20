http_port=$((RANDOM + 1024))
http_base_url=http://127.0.0.1:$http_port

python3 -m http.server -b 127.0.0.1 $http_port &> http.log &
trap "kill -9 $!" EXIT

i=0
while :; do
    python3 -c "from urllib.request import *; urlopen('$http_base_url')" &&
        break
    sleep 0.5
    if ((i++ >= 10)); then
        echo "timed out waiting for HTTP server"
        echo "http.log:"
        cat http.log
        exit 1
    fi
done
