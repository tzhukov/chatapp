
update_settings(k8s_upsert_timeout_secs=600)

load('ext://helm_resource', 'helm_resource','helm_repo')
load('ext://namespace','namespace_create','namespace_inject')

# Set namespace for all resources
namespace_create('chatapp')

helm_resource(
    name='kafka',
    chart='oci://registry-1.docker.io/bitnamicharts/kafka',
    namespace='chatapp',
    flags=['--set', 'kraft.enabled=true',
           '--set', 'zookeeper.enabled=true',
           '--set', 'listeners.client.protocol=PLAINTEXT']
)

# Deploy MongoDB from the Bitnami Helm chart
helm_resource(
    name='mongodb',
    chart='oci://registry-1.docker.io/bitnamicharts/mongodb',
    namespace='chatapp',
    flags=['--set', 'auth.enabled=false']
)

# Build the backend Docker image
docker_build('backend', 'backend')


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


# k8s_yaml('dex/ingress.yaml')

# Deploy the frontend Helm chart in chatapp namespace
k8s_yaml(helm('frontend/chart', name='frontend', namespace='chatapp'))

#Port forwards
k8s_resource(workload='frontend', port_forwards=[8081])
# k8s_resource(workload='dex', port_forwards=[9091])
# k8s_resource(workload='backend', port_forwards=[8082])
