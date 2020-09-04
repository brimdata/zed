print('Hello Tiltfile')

allow_k8s_contexts('zqdev')

def version():
  local('git describe --tags --dirty --always')

def ldflags():
    return '-s -X main.version=%s' % version()

docker_build(
    'localhost:5000/zqd',
    '.',
    build_args = { 'LDFLAGS': ldflags() },
    pull = True
)
