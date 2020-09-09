print('Hello Tiltfile')

allow_k8s_contexts('zqdev')

def version():
  local('git describe --tags --dirty --always')

def ldflags():
    return '-s -X main.version=%s' % version()

def local_strip_nl(command):
  return str(local(command)).rstrip('\n')

def aws_config(field):
  return local_strip_nl('aws configure get %s || true' % field)

def create_secret(name, namespace='', from_literal=None):
  args = [ 'kubectl', 'create', 'secret', 'generic', name ]
  if namespace:
    args.extend(['-n', namespace])
  for l in from_literal:
    args.extend(['--from-literal', l])
  args.extend(['-o=yaml', '--dry-run=client'])
  return local(args)

docker_build(
    'localhost:5000/zqd',
    '.',
    build_args = { 'LDFLAGS': ldflags() },
    pull = True
)

setValues = [
    'datauri=s3://zqd-demo-1/mark/zqd-meta'
]

# Values for EKS:
#    'AWSRegion="us-east-2"',
#    'image.repository="792043464098.dkr.ecr.us-east-2.amazonaws.com/"',

k8s_yaml(create_secret('aws-creds', 'zq', [
  'aws-access-key-id=' + aws_config('aws_access_key_id'),
  'aws-secret-access-key=' + aws_config('aws_secret_access_key'),
  'aws-session-token=' + aws_config('aws_session_token')
]))

k8s_yaml(helm(
    './charts/zqd',
    name='zqd',
    namespace='zq',
    set=setValues
))
