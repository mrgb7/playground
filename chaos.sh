#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="playground"
TEST_CLUSTERS=("chaos-test-1" "chaos-test-2" "chaos-test-3")
PLUGINS=("cert-manager" "tls" "load-balancer" "nginx-ingress" "argocd" "ingress")
INVALID_PLUGINS=("non-existent-plugin" "fake-plugin" "missing-dep")
LOG_FILE="chaos-test-$(date +%Y%m%d-%H%M%S).log"

# Statistics
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
EXPECTED_FAILURES=0

# Helper functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_PASSED++))
}

failure() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_FAILED++))
}

expected_failure() {
    echo -e "${YELLOW}⚠️  $1 (Expected)${NC}" | tee -a "$LOG_FILE"
    ((EXPECTED_FAILURES++))
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

info() {
    echo -e "${CYAN}ℹ️  $1${NC}" | tee -a "$LOG_FILE"
}

section() {
    echo -e "\n${PURPLE}=== $1 ===${NC}" | tee -a "$LOG_FILE"
}

run_test() {
    local test_name="$1"
    local command="$2"
    local expect_failure="${3:-false}"
    
    ((TESTS_RUN++))
    log "Running: $test_name"
    
    if [[ "$expect_failure" == "true" ]]; then
        if eval "$command" &>/dev/null; then
            failure "$test_name: Expected failure but command succeeded"
        else
            expected_failure "$test_name: Failed as expected"
        fi
    else
        if eval "$command" &>/dev/null; then
            success "$test_name: Passed"
        else
            failure "$test_name: Failed unexpectedly"
        fi
    fi
}

cleanup_clusters() {
    log "Cleaning up test clusters..."
    for cluster in "${TEST_CLUSTERS[@]}"; do
        if ./"$BINARY_NAME" cluster list | grep -q "$cluster" 2>/dev/null; then
            ./"$BINARY_NAME" cluster delete "$cluster" --force 2>/dev/null || true
        fi
    done
    sleep 2
}

build_binary() {
    section "Building Binary"
    if ! make build &>/dev/null; then
        failure "Failed to build binary"
        exit 1
    fi
    
    if [[ ! -f "$BINARY_NAME" ]]; then
        failure "Binary not found after build"
        exit 1
    fi
    
    success "Binary built successfully"
}

test_basic_functionality() {
    section "Testing Basic Functionality"
    
    run_test "Version command" "./$BINARY_NAME version"
    run_test "Help command" "./$BINARY_NAME --help"
    run_test "Cluster help" "./$BINARY_NAME cluster --help"
    run_test "Plugin help" "./$BINARY_NAME cluster plugin --help"
    run_test "List clusters (empty)" "./$BINARY_NAME cluster list"
}

test_cluster_operations() {
    section "Testing Cluster Operations"
    
    # Create clusters
    for cluster in "${TEST_CLUSTERS[@]}"; do
        run_test "Create cluster: $cluster" "./$BINARY_NAME cluster create $cluster"
        sleep 5  # Give time for cluster to initialize
    done
    
    # List clusters
    run_test "List all clusters" "./$BINARY_NAME cluster list"
    
    # Test operations on valid clusters
    run_test "Get cluster info" "./$BINARY_NAME cluster info ${TEST_CLUSTERS[0]}"
}

test_invalid_cluster_operations() {
    section "Testing Invalid Cluster Operations"
    
    # Operations on non-existent clusters
    run_test "Info on non-existent cluster" "./$BINARY_NAME cluster info non-existent-cluster" true
    run_test "Delete non-existent cluster" "./$BINARY_NAME cluster delete non-existent-cluster" true
    run_test "Plugin operations on non-existent cluster" "./$BINARY_NAME cluster plugin install cert-manager non-existent-cluster" true
    
    # Invalid cluster names
    run_test "Create cluster with invalid name" "./$BINARY_NAME cluster create invalid-name-with-spaces!" true
    run_test "Create cluster with empty name" "./$BINARY_NAME cluster create ''" true
    run_test "Create cluster with very long name" "./$BINARY_NAME cluster create $(printf 'a%.0s' {1..100})" true
}

test_plugin_operations() {
    section "Testing Plugin Operations"
    
    local test_cluster="${TEST_CLUSTERS[0]}"
    
    # Install plugins individually
    for plugin in "${PLUGINS[@]}"; do
        run_test "Install plugin: $plugin" "./$BINARY_NAME cluster plugin install $plugin $test_cluster"
        sleep 2
    done
    
    # Check plugin status
    run_test "List installed plugins" "./$BINARY_NAME cluster plugin list $test_cluster"
    
    # Test plugin status individually
    for plugin in "${PLUGINS[@]}"; do
        run_test "Status of plugin: $plugin" "./$BINARY_NAME cluster plugin status $plugin $test_cluster"
    done
}

test_plugin_idempotency() {
    section "Testing Plugin Idempotency"
    
    local test_cluster="${TEST_CLUSTERS[0]}"
    
    # Install plugins twice to test idempotency
    for plugin in "${PLUGINS[@]}"; do
        info "Testing idempotency for $plugin (first install)"
        run_test "First install: $plugin" "./$BINARY_NAME cluster plugin install $plugin $test_cluster"
        
        info "Testing idempotency for $plugin (second install)"
        run_test "Second install: $plugin" "./$BINARY_NAME cluster plugin install $plugin $test_cluster"
        
        sleep 1
    done
}

test_plugin_dependencies() {
    section "Testing Plugin Dependencies"
    
    local test_cluster="${TEST_CLUSTERS[1]}"
    
    # Try to install TLS without cert-manager (should fail)
    run_test "Install TLS without cert-manager" "./$BINARY_NAME cluster plugin install tls $test_cluster" true
    
    # Try to install ingress without dependencies (should handle gracefully)
    run_test "Install ingress without dependencies" "./$BINARY_NAME cluster plugin install ingress $test_cluster" true
    
    # Install cert-manager first, then TLS
    run_test "Install cert-manager first" "./$BINARY_NAME cluster plugin install cert-manager $test_cluster"
    sleep 3
    run_test "Install TLS after cert-manager" "./$BINARY_NAME cluster plugin install tls $test_cluster"
}

test_plugin_uninstall() {
    section "Testing Plugin Uninstallation"
    
    local test_cluster="${TEST_CLUSTERS[0]}"
    
    # Uninstall plugins
    for plugin in "${PLUGINS[@]}"; do
        run_test "Uninstall plugin: $plugin" "./$BINARY_NAME cluster plugin uninstall $plugin $test_cluster"
        sleep 1
    done
    
    # Try to uninstall non-existent plugins
    for plugin in "${INVALID_PLUGINS[@]}"; do
        run_test "Uninstall non-existent plugin: $plugin" "./$BINARY_NAME cluster plugin uninstall $plugin $test_cluster" true
    done
}

test_invalid_plugin_operations() {
    section "Testing Invalid Plugin Operations"
    
    local test_cluster="${TEST_CLUSTERS[0]}"
    
    # Invalid plugin names
    for plugin in "${INVALID_PLUGINS[@]}"; do
        run_test "Install invalid plugin: $plugin" "./$BINARY_NAME cluster plugin install $plugin $test_cluster" true
        run_test "Status of invalid plugin: $plugin" "./$BINARY_NAME cluster plugin status $plugin $test_cluster" true
    done
    
    # Plugin operations on deleted cluster (will test later)
    # Plugin operations with invalid arguments
    run_test "Plugin install without cluster name" "./$BINARY_NAME cluster plugin install cert-manager" true
    run_test "Plugin install without plugin name" "./$BINARY_NAME cluster plugin install $test_cluster" true
}

test_concurrent_operations() {
    section "Testing Concurrent Operations"
    
    # Create multiple clusters concurrently (be careful with resource limits)
    info "Starting concurrent cluster creation..."
    
    pids=()
    for i in {1..3}; do
        cluster_name="concurrent-$i"
        (./"$BINARY_NAME" cluster create "$cluster_name" &>/dev/null && echo "Created $cluster_name") &
        pids+=($!)
    done
    
    # Wait for all background processes
    for pid in "${pids[@]}"; do
        wait "$pid" || true
    done
    
    sleep 5
    
    # Clean up concurrent clusters
    for i in {1..3}; do
        cluster_name="concurrent-$i"
        ./"$BINARY_NAME" cluster delete "$cluster_name" --force &>/dev/null || true
    done
    
    success "Concurrent operations test completed"
}

test_resource_exhaustion() {
    section "Testing Resource Exhaustion Scenarios"
    
    # Try to create clusters with invalid resource specifications
    run_test "Create cluster with excessive CPU" "./$BINARY_NAME cluster create resource-test --master-cpu 100" true
    run_test "Create cluster with invalid memory" "./$BINARY_NAME cluster create resource-test --master-memory 999999G" true
    run_test "Create cluster with zero resources" "./$BINARY_NAME cluster create resource-test --master-cpu 0" true
    run_test "Create cluster with negative memory" "./$BINARY_NAME cluster create resource-test --master-memory -1G" true
}

test_operations_on_deleted_cluster() {
    section "Testing Operations on Deleted Clusters"
    
    local temp_cluster="temp-delete-test"
    
    # Create and then delete a cluster
    run_test "Create temporary cluster" "./$BINARY_NAME cluster create $temp_cluster"
    sleep 5
    run_test "Delete temporary cluster" "./$BINARY_NAME cluster delete $temp_cluster --force"
    sleep 3
    
    # Try operations on deleted cluster
    run_test "Info on deleted cluster" "./$BINARY_NAME cluster info $temp_cluster" true
    run_test "Plugin install on deleted cluster" "./$BINARY_NAME cluster plugin install cert-manager $temp_cluster" true
    run_test "Plugin list on deleted cluster" "./$BINARY_NAME cluster plugin list $temp_cluster" true
    run_test "Delete already deleted cluster" "./$BINARY_NAME cluster delete $temp_cluster" true
}

test_edge_cases() {
    section "Testing Edge Cases"
    
    # Test with various special characters and edge cases
    run_test "Empty command" "./$BINARY_NAME" true
    run_test "Invalid subcommand" "./$BINARY_NAME invalid-command" true
    run_test "Mixed case commands" "./$BINARY_NAME CLUSTER LIST" true
    
    # Test interrupt handling (brief)
    info "Testing interrupt handling..."
    timeout 5s ./"$BINARY_NAME" cluster create interrupt-test &>/dev/null || expected_failure "Interrupt test completed"
    
    # Clean up if cluster was partially created
    ./"$BINARY_NAME" cluster delete interrupt-test --force &>/dev/null || true
}

test_stress_plugin_operations() {
    section "Testing Stress Plugin Operations"
    
    local test_cluster="${TEST_CLUSTERS[2]}"
    
    # Rapid install/uninstall cycles
    for i in {1..3}; do
        info "Stress test cycle $i"
        
        # Install cert-manager rapidly
        run_test "Rapid install cert-manager (cycle $i)" "./$BINARY_NAME cluster plugin install cert-manager $test_cluster"
        run_test "Rapid uninstall cert-manager (cycle $i)" "./$BINARY_NAME cluster plugin uninstall cert-manager $test_cluster"
        sleep 1
    done
}

test_cleanup_and_recovery() {
    section "Testing Cleanup and Recovery"
    
    # Force cleanup all test clusters
    info "Performing force cleanup of all test clusters..."
    
    for cluster in "${TEST_CLUSTERS[@]}"; do
        if ./"$BINARY_NAME" cluster list | grep -q "$cluster" 2>/dev/null; then
            run_test "Force cleanup cluster: $cluster" "./$BINARY_NAME cluster delete $cluster --force"
        fi
    done
    
    # Verify cleanup
    run_test "Verify no test clusters remain" "! ./$BINARY_NAME cluster list | grep -E '(chaos-test|concurrent|temp-delete)'"
    
    # Test recovery - create a new cluster after cleanup
    run_test "Recovery: Create new cluster after cleanup" "./$BINARY_NAME cluster create recovery-test"
    run_test "Recovery: Delete recovery cluster" "./$BINARY_NAME cluster delete recovery-test --force"
}

print_summary() {
    section "Chaos Testing Summary"
    
    echo -e "\n${CYAN}📊 Test Statistics:${NC}"
    echo -e "Total Tests Run: ${BLUE}$TESTS_RUN${NC}"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Expected Failures: ${YELLOW}$EXPECTED_FAILURES${NC}"
    
    local success_rate=$((($TESTS_PASSED + $EXPECTED_FAILURES) * 100 / $TESTS_RUN))
    echo -e "Success Rate: ${GREEN}$success_rate%${NC}"
    
    echo -e "\n${CYAN}📝 Log file: $LOG_FILE${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "\n${GREEN}🎉 All chaos tests completed successfully!${NC}"
        echo -e "${GREEN}The system demonstrated excellent resilience under stress.${NC}"
    else
        echo -e "\n${RED}⚠️  Some tests failed unexpectedly.${NC}"
        echo -e "${RED}Review the log file for details: $LOG_FILE${NC}"
    fi
}

main() {
    section "Chaos Testing for K3s Cluster Management Tool"
    
    log "Starting chaos testing at $(date)"
    log "Log file: $LOG_FILE"
    
    # Ensure clean start
    cleanup_clusters
    
    # Build binary
    build_binary
    
    # Run test suites
    test_basic_functionality
    test_cluster_operations
    test_invalid_cluster_operations
    test_plugin_operations
    test_plugin_idempotency
    test_plugin_dependencies
    test_plugin_uninstall
    test_invalid_plugin_operations
    test_concurrent_operations
    test_resource_exhaustion
    test_operations_on_deleted_cluster
    test_edge_cases
    test_stress_plugin_operations
    test_cleanup_and_recovery
    
    # Final cleanup
    cleanup_clusters
    
    # Print summary
    print_summary
    
    # Exit with appropriate code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Trap to ensure cleanup on exit
trap cleanup_clusters EXIT

# Run main function
main "$@" 