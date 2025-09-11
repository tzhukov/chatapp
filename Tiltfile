update_settings(k8s_upsert_timeout_secs=600)

# NOTE: Removed Python try/except (unsupported in Starlark) that attempted to define TRIGGER_MODE_MANUAL.
# Tilt provides TRIGGER_MODE_MANUAL at runtime; IDE warnings can be ignored.

load('ext://helm_resource', 'helm_resource','helm_repo')
load('ext://namespace','namespace_create','namespace_inject')
# Built-in functions (docker_build, k8s_yaml, k8s_resource, local_resource, helm) are provided by Tilt automatically.

# Set namespace for all resources
namespace_create('chatapp')

helm_resource(
    name='kafka',
    chart='oci://registry-1.docker.io/bitnamicharts/kafka',
    namespace='chatapp',
    flags=['--set', 'kraft.enabled=true',
           '--set', 'zookeeper.enabled=true',
           '--set', 'listeners.client.protocol=PLAINTEXT',
           '--set','resources.limits.cpu=500m',
           '--set','resources.limits.memory=512Mi',
           '--set','resources.requests.cpu=200m',
           '--set','resources.requests.memory=256Mi']
)

k8s_yaml('mongo/pvc.yaml')


# Deploy MongoDB from the Bitnami Helm chart
helm_resource(
    name='mongodb',
    chart='oci://registry-1.docker.io/bitnamicharts/mongodb',
    namespace='chatapp',
    flags=[
        '--set','auth.enabled=false',
    '--set','persistence.enabled=true',
    '--set','persistence.existingClaim=mongodb',
        '--set','resources.limits.cpu=500m',
        '--set','resources.limits.memory=512Mi',
        '--set','resources.requests.cpu=200m',
        '--set','resources.requests.memory=256Mi'
    ]
)

# Build the backend Docker image
docker_build('backend', 'backend')

# Backend unit tests (runs `go test ./...`) without blocking deploys; depends on source changes.
local_resource(
    name='backend-tests',
    cmd='bash scripts/test_backend.sh',
    deps=['backend/src','backend/src/go.mod','backend/src/go.sum'],
    labels=['tests','backend'],
    # Use Tilt's TriggerMode constant instead of an invalid string to fix Tiltfile error
    # Manual trigger (Tilt built-in constant)
    trigger_mode=TRIGGER_MODE_MANUAL
)


# Deploy the backend Helm chart in chatapp namespace
k8s_yaml(helm('backend/chart', name='backend', namespace='chatapp'))

# Deploy Keycloak from the public Helm chart


# Deploy Authelia from the public Helm chart

helm_repo('dex-repo', 'https://charts.dexidp.io')


# Deploy Authelia from the official Helm chart repo


helm_resource(
    name='dex',
    chart='dex-repo/dex',
    resource_deps=['dex-repo'],
    namespace='chatapp',
    flags=['--values=./dex/dex-values.yaml'],
    )

# Build the frontend Docker image
docker_build('frontend', 'frontend')

# Frontend unit/integration tests (Jest). Assumes dependencies installed; install if node_modules missing.
local_resource(
    name='frontend-tests',
    cmd='bash scripts/test_frontend.sh',
    deps=['frontend/src','frontend/tests','frontend/package.json','frontend/jest.config.js','frontend/babel.config.js'],
    labels=['tests','frontend'],
    trigger_mode=TRIGGER_MODE_MANUAL
)



# Deploy the frontend Helm chart in chatapp namespace
k8s_yaml(helm('frontend/chart', name='frontend', namespace='chatapp'))

#Port forwards
k8s_resource(workload='frontend', port_forwards=[8081])
# k8s_resource(workload='dex', port_forwards=[9091])
# k8s_resource(workload='backend', port_forwards=[8082])
