#!/bin/bash

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'
PLAYGROUND_BIN="./playground"
TEST_CLUSTER_PREFIX="e2e-dep-test"
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo ""
}


cleanup_cluster() {
    local cluster_name=$1
    log_info "Cleaning up cluster: $cluster_name"
    if $PLAYGROUND_BIN cluster delete $cluster_name &>/dev/null; then
        log_success "Cluster $cluster_name deleted successfully"
    else
        log_warning "Cluster $cluster_name may not exist or already deleted"
    fi
}


wait_for_cluster() {
    local cluster_name=$1
    log_info "Waiting for cluster $cluster_name to be ready..."
    sleep 10
}


expect_failure() {
    local description=$1
    shift
    log_info "Expecting failure: $description"
    
    if "$@" &>/dev/null; then
        log_error "Command should have failed but succeeded: $description"
        return 1
    else
        log_success "Command failed as expected: $description"
        return 0
    fi
}


expect_success() {
    local description=$1
    shift
    log_info "Expecting success: $description"
    
    if "$@"; then
        log_success "Command succeeded as expected: $description"
        return 0
    else
        log_error "Command should have succeeded but failed: $description"
        return 1
    fi
}


show_dependencies() {
    local cluster_name=$1
    log_info "Plugin dependency information for $cluster_name:"
    $PLAYGROUND_BIN cluster plugin deps --cluster $cluster_name || log_warning "Failed to show dependencies"
}


show_plugin_status() {
    local cluster_name=$1
    log_info "Plugin status for $cluster_name:"
    $PLAYGROUND_BIN cluster plugin list --cluster-name $cluster_name || log_warning "Failed to show plugin status"
}


test_happy_path() {
    log_section "TEST 1: Happy Path - Install plugins in correct dependency order"
    
    local cluster_name="${TEST_CLUSTER_PREFIX}-happy"
    
    cleanup_cluster $cluster_name
    log_info "Creating cluster: $cluster_name"
    expect_success "Cluster creation" $PLAYGROUND_BIN cluster create --name $cluster_name
    wait_for_cluster $cluster_name
    
    show_dependencies $cluster_name
    show_plugin_status $cluster_name
    log_info "Installing cert-manager (no dependencies)"
    expect_success "Install cert-manager" $PLAYGROUND_BIN cluster plugin add --name cert-manager --cluster $cluster_name
    
    log_info "Installing load-balancer (no dependencies)"
    expect_success "Install load-balancer" $PLAYGROUND_BIN cluster plugin add --name load-balancer --cluster $cluster_name
    
    log_info "Installing nginx-ingress (depends on load-balancer)"
    expect_success "Install nginx-ingress" $PLAYGROUND_BIN cluster plugin add --name nginx-ingress --cluster $cluster_name
    
    log_info "Installing tls (depends on cert-manager)"
    expect_success "Install tls" $PLAYGROUND_BIN cluster plugin add --name tls --cluster $cluster_name
    
    log_info "Installing ingress (depends on nginx-ingress and load-balancer)"
    expect_success "Install ingress" $PLAYGROUND_BIN cluster plugin add --name ingress --cluster $cluster_name
    
    log_info "Final plugin status after installations:"
    show_plugin_status $cluster_name
    cleanup_cluster $cluster_name
    log_success "TEST 1 COMPLETED: Happy path installation test passed!"
}


test_dependency_error() {
    log_section "TEST 2: Dependency Error - Install plugin before its dependencies"
    
    local cluster_name="${TEST_CLUSTER_PREFIX}-dep-error"
    
    cleanup_cluster $cluster_name
    
    log_info "Creating cluster: $cluster_name"
    expect_success "Cluster creation" $PLAYGROUND_BIN cluster create --name $cluster_name
    wait_for_cluster $cluster_name
    
    show_dependencies $cluster_name
    show_plugin_status $cluster_name
    
    log_info "Attempting to install ingress without dependencies (should fail)"
    expect_failure "Install ingress without dependencies" $PLAYGROUND_BIN cluster plugin add --name ingress --cluster $cluster_name
    
    log_info "Attempting to install nginx-ingress without load-balancer (should fail)"
    expect_failure "Install nginx-ingress without load-balancer" $PLAYGROUND_BIN cluster plugin add --name nginx-ingress --cluster $cluster_name
    
    log_info "Attempting to install tls without cert-manager (should fail)"
    expect_failure "Install tls without cert-manager" $PLAYGROUND_BIN cluster plugin add --name tls --cluster $cluster_name
    
    log_info "Plugin status after failed installation attempts:"
    show_plugin_status $cluster_name
    
    log_info "Installing load-balancer first"
    expect_success "Install load-balancer" $PLAYGROUND_BIN cluster plugin add --name load-balancer --cluster $cluster_name
    
    log_info "Now installing nginx-ingress (should work)"
    expect_success "Install nginx-ingress after load-balancer" $PLAYGROUND_BIN cluster plugin add --name nginx-ingress --cluster $cluster_name
    
    log_info "Final plugin status:"
    show_plugin_status $cluster_name
    cleanup_cluster $cluster_name
    log_success "TEST 2 COMPLETED: Dependency error test passed!"
}


test_removal_error() {
    log_section "TEST 3: Removal Error - Try to delete plugin that others depend on"
    
    local cluster_name="${TEST_CLUSTER_PREFIX}-removal-error"
    
    cleanup_cluster $cluster_name
    
    log_info "Creating cluster: $cluster_name"
    expect_success "Cluster creation" $PLAYGROUND_BIN cluster create --name $cluster_name
    wait_for_cluster $cluster_name
    
    log_info "Installing all plugins in correct dependency order"
    expect_success "Install cert-manager" $PLAYGROUND_BIN cluster plugin add --name cert-manager --cluster $cluster_name
    expect_success "Install load-balancer" $PLAYGROUND_BIN cluster plugin add --name load-balancer --cluster $cluster_name
    expect_success "Install nginx-ingress" $PLAYGROUND_BIN cluster plugin add --name nginx-ingress --cluster $cluster_name
    expect_success "Install tls" $PLAYGROUND_BIN cluster plugin add --name tls --cluster $cluster_name
    expect_success "Install ingress" $PLAYGROUND_BIN cluster plugin add --name ingress --cluster $cluster_name
    
    log_info "All plugins installed, current status:"
    show_plugin_status $cluster_name
    
    log_info "Attempting to remove load-balancer (should fail - has dependents)"
    expect_failure "Remove load-balancer with dependents" $PLAYGROUND_BIN cluster plugin remove --name load-balancer --cluster $cluster_name
    
    log_info "Attempting to remove nginx-ingress (should fail - ingress depends on it)"
    expect_failure "Remove nginx-ingress with dependents" $PLAYGROUND_BIN cluster plugin remove --name nginx-ingress --cluster $cluster_name
    
    log_info "Attempting to remove cert-manager (should fail - tls depends on it)"
    expect_failure "Remove cert-manager with dependents" $PLAYGROUND_BIN cluster plugin remove --name cert-manager --cluster $cluster_name
    
    log_info "Removing ingress first (no dependents)"
    expect_success "Remove ingress" $PLAYGROUND_BIN cluster plugin remove --name ingress --cluster $cluster_name
    
    log_info "Now removing nginx-ingress (no longer has dependents)"
    expect_success "Remove nginx-ingress" $PLAYGROUND_BIN cluster plugin remove --name nginx-ingress --cluster $cluster_name
    
    log_info "Removing tls (no dependents)"
    expect_success "Remove tls" $PLAYGROUND_BIN cluster plugin remove --name tls --cluster $cluster_name
    
    log_info "Now removing cert-manager (no longer has dependents)"
    expect_success "Remove cert-manager" $PLAYGROUND_BIN cluster plugin remove --name cert-manager --cluster $cluster_name
    
    log_info "Finally removing load-balancer (no longer has dependents)"
    expect_success "Remove load-balancer" $PLAYGROUND_BIN cluster plugin remove --name load-balancer --cluster $cluster_name
    
    log_info "Final plugin status after all removals:"
    show_plugin_status $cluster_name
    cleanup_cluster $cluster_name
    log_success "TEST 3 COMPLETED: Removal error test passed!"
}


test_multiple_plugins() {
    log_section "TEST 4: Multiple Plugin Installation and Removal"
    
    local cluster_name="${TEST_CLUSTER_PREFIX}-multiple"
    
    cleanup_cluster $cluster_name
    
    log_info "Creating cluster: $cluster_name"
    expect_success "Cluster creation" $PLAYGROUND_BIN cluster create --name $cluster_name
    wait_for_cluster $cluster_name
    
    log_info "Installing ingress (should automatically install dependencies)"
    expect_success "Install ingress with auto-dependencies" $PLAYGROUND_BIN cluster plugin add --name ingress --cluster $cluster_name
    
    log_info "Status after installing ingress (should show dependencies installed):"
    show_plugin_status $cluster_name
    
    log_info "Installing remaining plugins"
    expect_success "Install cert-manager" $PLAYGROUND_BIN cluster plugin add --name cert-manager --cluster $cluster_name
    expect_success "Install tls" $PLAYGROUND_BIN cluster plugin add --name tls --cluster $cluster_name
    expect_success "Install argocd" $PLAYGROUND_BIN cluster plugin add --name argocd --cluster $cluster_name
    
    log_info "Final status with all plugins:"
    show_plugin_status $cluster_name
    
    log_info "Removing multiple plugins with dependencies"
    expect_success "Remove ingress, nginx-ingress, load-balancer" $PLAYGROUND_BIN cluster plugin remove --name "ingress,nginx-ingress,load-balancer" --cluster $cluster_name
    
    log_info "Status after multiple plugin removal:"
    show_plugin_status $cluster_name
    cleanup_cluster $cluster_name
    log_success "TEST 4 COMPLETED: Multiple plugin test passed!"
}


main() {
    log_section "Starting E2E Dependency Graph Testing"

    if [[ ! -f "$PLAYGROUND_BIN" ]]; then
        log_error "Playground binary not found at $PLAYGROUND_BIN"
        log_info "Please run: go build -o playground ."
        exit 1
    fi

    log_info "Cleaning up any existing test clusters..."
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-happy" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-dep-error" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-removal-error" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-multiple" || true
    
    local start_time=$(date +%s)
    local tests_passed=0
    local tests_failed=0

    if test_happy_path; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_dependency_error; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_removal_error; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_multiple_plugins; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "Final cleanup..."
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-happy" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-dep-error" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-removal-error" || true
    cleanup_cluster "${TEST_CLUSTER_PREFIX}-multiple" || true

    log_section "E2E Testing Summary"
    log_info "Tests passed: $tests_passed"
    log_info "Tests failed: $tests_failed"
    log_info "Total duration: ${duration} seconds"
    
    if [[ $tests_failed -eq 0 ]]; then
        log_success "🎉 All E2E dependency tests passed!"
        exit 0
    else
        log_error "❌ Some E2E dependency tests failed!"
        exit 1
    fi
}


trap 'log_warning "Script interrupted, cleaning up..."; cleanup_cluster "${TEST_CLUSTER_PREFIX}-happy" || true; cleanup_cluster "${TEST_CLUSTER_PREFIX}-dep-error" || true; cleanup_cluster "${TEST_CLUSTER_PREFIX}-removal-error" || true; cleanup_cluster "${TEST_CLUSTER_PREFIX}-multiple" || true; exit 1' INT TERM


main "$@" 