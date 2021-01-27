function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for file \"$file\" to appear"
      echo "zqd log:"
      cat zqd.log
      exit 1
    fi
    sleep 1
  done
}

pgurl=${PG_TEST:='postgres://test:test@localhost:5432/test?sslmode=disable'}
dbname=$(pgctl testdb -p $pgurl -m ./migrations)
portdir=$(mktemp -d)
mkdir -p data

zqd listen -l=localhost:0 \
  -data=data \
  -loglevel=warn \
  -portfile="$portdir/zqd" \
  -db.kind="postgres" \
  -db.postgres.url=$PG_TEST \
  -db.postgres.database=$dbname $ZQD_EXTRA_FLAGS &> zqd.log &

zqdpid=$!
awaitfile $portdir/zqd
trap "rm -rf $portdir ; kill -9 $zqdpid &>/dev/null ; pgctl rmtestdb -p $PG_TEST $dbname ;" EXIT

export ZQD_HOST=localhost:$(cat $portdir/zqd)
