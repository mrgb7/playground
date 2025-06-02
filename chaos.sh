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

# Check observability plugin is added
if kubectl get namespace monitoring > /dev/null 2>&1; then
    echo "✅ Observability 'monitoring' namespace found."
else
    echo "❌ Observability 'monitoring' namespace not found. Test failed."
    exit 1
fi

# Wait for observability components to be ready (Victoria Metrics)
echo "Waiting for Victoria Metrics to be ready..."
if kubectl rollout status -n monitoring statefulset/vmsingle-observability --timeout=300s > /dev/null 2>&1; then
    echo "✅ Victoria Metrics is ready."
else
    echo "❌ Victoria Metrics rollout status check failed. Continuing with other checks..."
fi

# Wait for Grafana to be ready
echo "Waiting for Grafana to be ready..."
if kubectl rollout status -n monitoring deployment/grafana --timeout=300s > /dev/null 2>&1; then
    echo "✅ Grafana is ready."
else
    echo "❌ Grafana rollout status check failed. Continuing with other checks..."
fi

# Check if observability components are running
echo "Checking observability components..."
if kubectl get pods -n monitoring | grep -q "Running"; then
    echo "✅ Observability components are running."
else
    echo "❌ Observability components are not running properly. Test failed."
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

# Check if observability ingress resources are created
sleep 30
if kubectl get ingress -n monitoring grafana-ingress > /dev/null 2>&1; then
    echo "✅ Grafana ingress found."
else
    echo "❌ Grafana ingress not found. Ingress integration may have issues."
fi

if kubectl get ingress -n monitoring victoria-metrics-ingress > /dev/null 2>&1; then
    echo "✅ Victoria Metrics ingress found."
else
    echo "❌ Victoria Metrics ingress not found. Ingress integration may have issues."
fi

# Remove observability plugin
echo "Removing observability plugin from the cluster..."
if ./bin/playground cluster plugin remove -n observability -c chaos-test > /dev/null 2>&1; then
    echo "✅ Observability plugin removed successfully."
else
    echo "❌ Failed to remove observability plugin. Exiting."
    exit 1
fi

sleep 10
# Check observability plugin is removed
kubectl get namespace monitoring > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "❌ Observability 'monitoring' namespace still exists. Test failed."
    exit 1
else
    echo "✅ Observability plugin removed successfully."
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
