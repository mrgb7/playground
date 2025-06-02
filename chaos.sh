#!/bin/sh

function cleanup {
    echo "Cleaning up chaos-test cluster..."
    ERR=$(./bin/playground cluster delete chaos-test)
    if [ $? -ne 0 ]; then
        if echo "$ERR" | grep -q "does not exist."; then
            echo "✅ Chaos-test cluster deleted successfully."
        else 
            echo "❌ Chaos-test cluster deletion failed unexpectedly. Test failed."
            exit 1
        fi
    else
        echo "✅ Chaos-test cluster deleted successfully."
   fi
}

echo "Starting chaos testing..."

if make build > /dev/null 2>&1; then
    echo "✅ Build successful."
else
    echo "❌ Build failed. Exiting."
    exit 1
fi

echo "✅ Build successful."

cleanup



playground cluster list > /dev/null
if [ $? -ne 0 ]; then
    echo "❌ Failed to list clusters. Exiting."
    exit 1
fi

# Create a cluster named "chaos-test" with the default configuration
./bin/playground cluster create --name chaos-test
if [ $? -ne 0 ]; then
    echo "❌ Failed to create cluster. Exiting."
    exit 1
fi

# List clusters to verify creation
./bin/playground cluster List > /dev/null
if [ $? -ne 0 ]; then
    echo "❌ Failed to list clusters after creation. Exiting."
    exit 1
fi

# Create again to check if it fails

CREATE_DUPLICATE_TEST=$(./bin/playground cluster create --name chaos-test)

if echo "$CREATE_DUPLICATE_TEST" | grep -q "Failed to create cluster:"; then
    echo "✅ Cluster creation failed as expected. Test passed."
else
    echo "❌ Cluster creation did not fail as expected. Test failed."
    exit 1
fi

kubectl config use-context chaos-test-context > /dev/null
if [ $? -ne 0 ]; then
    echo "❌ Failed to switch context to chaos-test. Exiting."
    exit 1
fi

echo "Switched context to chaos-test successfully."
# Add the ArgoCD plugin to the clusters
echo "Adding ArgoCD plugin to the cluster..."
if ./bin/playground cluster plugin add -n argocd -c chaos-test > /dev/null 2>&1; then
    echo "✅ ArgoCD plugin added successfully."
else
    echo "❌ Failed to add ArgoCD plugin. Exiting."
    exit 1
fi


#Check argocd plugin is added

if kubectl get namespace argocd > /dev/null 2>&1; then
    echo "✅ ArgoCD namespace found."
else
    echo "❌ ArgoCD namespace not found. Test failed."
    exit 1
fi


# Wait for ArgoCD to be ready
echo "Waiting for ArgoCD to be ready..."
if kubectl rollout status -n argocd deployment/argocd-server --timeout=300s > /dev/null 2>&1; then
    echo "✅ ArgoCD is ready."
else
    echo "❌ ArgoCD rollout status check failed. Exiting."
    exit 1
fi

# Check Install Loadbalancer plugin

echo "Adding LoadBalancer plugin to the cluster..."
if ./bin/playground cluster plugin add -n load-balancer -c chaos-test >> /dev/null; then
    echo "✅ LoadBalancer plugin added successfully."
else
    echo "❌ Failed to add LoadBalancer plugin. Exiting."
    exit 1
fi
# Check LoadBalancer plugin is added
if kubectl get namespace metallb-system > /dev/null 2>&1; then
    echo "✅ LoadBalancer 'metallb-system' namespace found."
else
    echo "❌ LoadBalancer 'metallb-system' namespace not found. Test failed."
    exit 1
fi
# Argocd deletion should fail because LB plugin is installed by argocd

output=$(./bin/playground cluster plugin remove -n argocd -c chaos-test 2>&1)
if echo "$output" | grep -q "you cannot uninstall argocd because it is used by other plugins:"; then
    echo "✅ Plugin deletion failed as expected. Test passed."
else
    echo "❌ Plugin deletion did not fail as expected. Test failed."
    exit 1
fi
# Remove LoadBalancer plugins
# echo "Removing LoadBalancer plugin from the cluster..."
if ./bin/playground cluster plugin remove -n load-balancer -c chaos-test > /dev/null 2>&1; then
    echo "✅ LoadBalancer plugin removed successfully."
else
    echo "❌ Failed to remove LoadBalancer plugin. Exiting."
    exit 1
fi

# Check LoadBalancer plugin is removed
kubectl get namespace metallb-system > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "❌ LoadBalancer 'metallb-system' namespace still exists. Test failed."
    exit 1
else
    echo "✅ LoadBalancer plugin removed successfully."
fi
# Remove ArgoCD plugin it should succeed now
echo "Removing ArgoCD plugin from the cluster..."
if ./bin/playground cluster plugin remove -n argocd -c chaos-test > /dev/null 2>&1; then
    echo "✅ ArgoCD plugin removed successfully."
else
    echo "❌ Failed to remove ArgoCD plugin. Exiting."
    exit 1
fi


# Check ArgoCD plugin is removed
kubectl get namespace argocd > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "❌ ArgoCD namespace still exists. Test failed."
    exit 1
else
    echo "✅ ArgoCD plugin removed successfully."
fi

# Test dependency Check

echo "Installing TLS plugin without cert-manager..."
output=$(./bin/playground cluster plugin add -n tls -c chaos-test 2>&1)

# It should install cert-manager as a dependency
 kubectl get namespace cert-manager > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ cert-manager namespace not found. Test failed."
    exit 1
else
    echo "✅ cert-manager namespace found. Test passed."
fi

# Remove TLS plugin
echo "Removing TLS plugin from the cluster..."
if ./bin/playground cluster plugin remove -n tls -c chaos-test > /dev/null 2>&1; then
    echo "✅ TLS plugin removed successfully."
else
    echo "❌ Failed to remove TLS plugin. Exiting."
    exit 1
fi

sleep 10
# Cert-manager should not be impacted by TLS plugin removal
if kubectl get namespace cert-manager > /dev/null 2>&1 ; then
    echo "✅ cert-manager namespace still exists after TLS plugin removal."
else
    echo "❌ cert-manager namespace not found after TLS plugin removal. Test failed."
    exit 1
fi

# Remove cert-manager plugins
echo "Removing cert-manager plugin from the cluster..."
if ./bin/playground cluster plugin remove -n cert-manager -c chaos-test > /dev/null 2>&1; then
    echo "✅ cert-manager plugin removed successfully."
else
    echo "❌ Failed to remove cert-manager plugin. Exiting."
    exit 1
fi
sleep 10
# Check cert-manager plugin is removed
if kubectl get namespace cert-manager > /dev/null 2>&1 ; then
    echo "❌ cert-manager namespace still exists. Test failed."
    exit 1
else
    echo "✅ cert-manager plugin removed successfully."
fi

# Test Observability Plugin
echo "Testing observability plugin..."
echo "Installing observability plugin..."
if ./bin/playground cluster plugin add -n observability -c chaos-test > /dev/null 2>&1; then
    echo "✅ Observability plugin added successfully."
else
    echo "❌ Failed to add observability plugin. Exiting."
    exit 1
fi

# Wait for namespace to be created
echo "Waiting for monitoring namespace to be created..."
retry_count=0
max_retries=30
while [ $retry_count -lt $max_retries ]; do
    if kubectl get namespace monitoring > /dev/null 2>&1; then
        echo "✅ Observability 'monitoring' namespace found."
        break
    else
        echo "⏳ Waiting for monitoring namespace... (attempt $((retry_count + 1))/$max_retries)"
        sleep 10
        retry_count=$((retry_count + 1))
    fi
done

if [ $retry_count -eq $max_retries ]; then
    echo "❌ Observability 'monitoring' namespace not found after $max_retries attempts. Test failed."
    exit 1
fi

# Wait for all observability components to be deployed
echo "Waiting for observability components to be deployed..."
sleep 30

# Check specific deployments are created with correct names
echo "Checking Grafana deployment..."
kubectl get deployment -n monitoring observability-grafana > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Grafana Deployment found."
else
    echo "❌ Grafana Deployment not found."
fi

echo "Checking Node Exporter DaemonSet..."
kubectl get daemonset -n monitoring observability-prometheus-node-exporter > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Node Exporter DaemonSet found."
else
    echo "❌ Node Exporter DaemonSet not found."
fi

echo "Checking Kube State Metrics deployment..."
kubectl get deployment -n monitoring observability-kube-state-metrics > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Kube State Metrics Deployment found."
else
    echo "❌ Kube State Metrics Deployment not found."
fi

echo "Checking Victoria Metrics Operator deployment..."
kubectl get deployment -n monitoring observability-victoria-metrics-operator > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Victoria Metrics Operator Deployment found."
else
    echo "❌ Victoria Metrics Operator Deployment not found."
fi

echo "Checking Victoria Metrics Single deployment..."
kubectl get deployment -n monitoring vmsingle-observability-victoria-metrics-k8s-stack > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Victoria Metrics Single Deployment found."
else
    echo "❌ Victoria Metrics Single Deployment not found."
fi

echo "Checking Victoria Metrics Agent deployment..."
kubectl get deployment -n monitoring vmagent-observability-victoria-metrics-k8s-stack > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Victoria Metrics Agent Deployment found."
else
    echo "❌ Victoria Metrics Agent Deployment not found."
fi

# Wait for core observability components to be ready
echo "Waiting for Grafana to be ready..."
if kubectl rollout status -n monitoring deployment/observability-grafana --timeout=300s > /dev/null 2>&1; then
    echo "✅ Grafana is ready."
else
    echo "⚠️  Grafana rollout status check failed. Continuing with other checks..."
fi

echo "Waiting for Node Exporter to be ready..."
if kubectl rollout status -n monitoring daemonset/observability-prometheus-node-exporter --timeout=300s > /dev/null 2>&1; then
    echo "✅ Node Exporter is ready."
else
    echo "⚠️  Node Exporter rollout status check failed. Continuing with other checks..."
fi

echo "Waiting for Kube State Metrics to be ready..."
if kubectl rollout status -n monitoring deployment/observability-kube-state-metrics --timeout=300s > /dev/null 2>&1; then
    echo "✅ Kube State Metrics is ready."
else
    echo "⚠️  Kube State Metrics rollout status check failed. Continuing with other checks..."
fi

echo "Waiting for Victoria Metrics Operator to be ready..."
if kubectl rollout status -n monitoring deployment/observability-victoria-metrics-operator --timeout=300s > /dev/null 2>&1; then
    echo "✅ Victoria Metrics Operator is ready."
else
    echo "⚠️  Victoria Metrics Operator rollout status check failed. Continuing with other checks..."
fi

echo "Waiting for Victoria Metrics Single to be ready..."
if kubectl rollout status -n monitoring deployment/vmsingle-observability-victoria-metrics-k8s-stack --timeout=300s > /dev/null 2>&1; then
    echo "✅ Victoria Metrics Single is ready."
else
    echo "⚠️  Victoria Metrics Single rollout status check failed. Continuing with other checks..."
fi

echo "Waiting for Victoria Metrics Agent to be ready..."
if kubectl rollout status -n monitoring deployment/vmagent-observability-victoria-metrics-k8s-stack --timeout=300s > /dev/null 2>&1; then
    echo "✅ Victoria Metrics Agent is ready."
else
    echo "⚠️  Victoria Metrics Agent rollout status check failed. Continuing with other checks..."
fi

# Check if observability components are running
echo "Checking if observability pods are running..."
sleep 15
running_pods=$(kubectl get pods -n monitoring --no-headers | grep -c "Running")
total_pods=$(kubectl get pods -n monitoring --no-headers | wc -l)

if [ "$running_pods" -gt 0 ]; then
    echo "✅ Observability components are running ($running_pods/$total_pods pods in Running state)."
    
    # List all pods for debugging
    echo "Observability pods status:"
    kubectl get pods -n monitoring
else
    echo "❌ No observability components are running properly."
    echo "Pod status for debugging:"
    kubectl get pods -n monitoring
    echo "❌ Observability components test failed."
    exit 1
fi

# Test ingress integration with observability
echo "Testing ingress integration with observability..."
echo "Installing ingress plugin to test observability integration..."
if ./bin/playground cluster plugin add -n ingress -c chaos-test > /dev/null 2>&1; then
    echo "✅ Ingress plugin added successfully."
else
    echo "❌ Failed to add ingress plugin. Continuing without ingress tests..."
fi

# Wait for ingress to create observability ingresses
echo "Waiting for observability ingresses to be created..."
sleep 45

# Check if observability ingress resources are created
echo "Checking for Grafana ingress..."
if kubectl get ingress -n monitoring grafana-ingress > /dev/null 2>&1; then
    echo "✅ Grafana ingress found."
else
    echo "⚠️  Grafana ingress not found. Ingress integration may need more time."
fi

# List all ingresses in monitoring namespace for debugging
echo "All ingresses in monitoring namespace:"
kubectl get ingress -n monitoring

# Remove observability plugin
echo "Removing observability plugin from the cluster..."
if ./bin/playground cluster plugin remove -n observability -c chaos-test > /dev/null 2>&1; then
    echo "✅ Observability plugin removed successfully."
else
    echo "❌ Failed to remove observability plugin. Exiting."
    exit 1
fi

# Wait for cleanup
sleep 15

# Check observability plugin is removed
retry_count=0
max_retries=20
while [ $retry_count -lt $max_retries ]; do
    if kubectl get namespace monitoring > /dev/null 2>&1; then
        echo "⏳ Waiting for monitoring namespace to be deleted... (attempt $((retry_count + 1))/$max_retries)"
        sleep 15
        retry_count=$((retry_count + 1))
    else
        echo "✅ Observability plugin removed successfully."
        break
    fi
done

if [ $retry_count -eq $max_retries ]; then
    echo "⚠️  Observability 'monitoring' namespace still exists after $max_retries attempts. May take longer to clean up."
fi

# Clean up ingress plugin if it was installed
echo "Cleaning up ingress plugin..."
./bin/playground cluster plugin remove -n ingress -c chaos-test > /dev/null 2>&1

# Install ingress plugin with NGINX controller and cert-manager as dependencies
echo "Installing Ingress plugin with NGINX controller and cert-manager as dependencies..."
if ./bin/playground cluster plugin add -n ingress -c chaos-test > /dev/null 2>&1; then
    echo "✅ Ingress plugin installed successfully."
else
    echo "❌ Failed to install Ingress plugin. Exiting."
    exit 1
fi
sleep 60
# Check Ingress plugin is installed
if kubectl get namespace ingress-nginx > /dev/null 2>&1; then
    echo "✅ Ingress NGINX namespace found. ingress plugin dependencies are installed."
else
    echo "❌ Ingress NGINX namespace not found. Test failed. ingress plugin dependencies are not installed."
    exit 1
fi
# Check metallb is installed as a dependency
if kubectl get namespace metallb-system > /dev/null 2>&1; then
    echo "✅ load-balancer namespace found. Ingress plugin dependencies are installed."
else
    echo "❌ load-balancer namespace not found. Test failed. Ingress plugin dependencies are not installed."
    exit 1
fi
# clean up any existing chaos-test cluster
cleanup

# Create a new chaos-test cluster with Specified configuration
if ./bin/playground cluster create --name chaos-test --size 2 --master-cpus 1  --master-memory 1G --worker-cpus 1 --worker-memory 1G > /dev/null 2>&1; then
    echo "✅ Chaos-test cluster created successfully with specified configuration."
else
    echo "❌ Failed to create chaos-test cluster with specified configuration. Exiting."
    exit 1
fi

cleanup
