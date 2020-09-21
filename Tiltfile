# This assumes you have created a kubctl config context
# called `zqtest` before running Tilt.

# If the build needs to create a new namespace use:
# load('ext://namespace', 'namespace_yaml')
# k8s_yaml(namespace_yaml('zq'))

allow_k8s_contexts('zqtest')

def local_strip_nl(command):
  return str(local(command)).rstrip('\n')

def version():
  return local_strip_nl('git describe --tags --dirty --always')

def zqd_version():
  return 'zqd-%s' % version()

def ldflags():
  return '-s -X main.version=%s' % version()

def current_context():
  return local_strip_nl('kubectl config current-context')

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

if current_context() != 'zqtest':
  registry_prefix='localhost:5000/'
  k8s_yaml(create_secret('aws-creds', 'zq', [
    'aws-access-key-id=' + aws_config('aws_access_key_id'),
    'aws-secret-access-key=' + aws_config('aws_secret_access_key'),
    'aws-session-token=' + aws_config('aws_session_token')
  ]))
  setValues = [
    'datauri=s3://zqd-demo-1/mark/zqd-meta',
    'AWSRegion="us-east-2"',
    'image.tag=' + zqd_version()
  ]
else:
  registry_prefix='792043464098.dkr.ecr.us-east-2.amazonaws.com/'
  # Log in to registry
  # local('aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin 792043464098.dkr.ecr.us-east-2.amazonaws.com/zqd')
  setValues = [
    'datauri=s3://zqd-demo-1/mark/zqd-meta',
    'AWSRegion=us-east-2',
    'image.repository=792043464098.dkr.ecr.us-east-2.amazonaws.com/',
    'image.tag=' + zqd_version(),
    'useCredSecret=false'
  ]

docker_build(
    registry_prefix + zqd_version(),
    '.',
    build_args = { 'LDFLAGS': ldflags() },
    pull = True
)

k8s_yaml(helm(
    './charts/zqd',
    name='zqd',
    set=setValues
))

