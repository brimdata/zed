kubectl port-forward svc/zsrv-recruiter 8020:9867 &
kubectl port-forward svc/zsrv-root 9867:9867 &
for i in {1..5}; do
  if nc -z localhost 8020 ; then
    break
  fi
  sleep 1
done
for i in {1..5}; do
  if nc -z localhost 9867 ; then
    break
  fi
  sleep 1
done
