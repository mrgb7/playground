# Chaos Testing Script

## Overview

The `chaos.sh` script is a comprehensive chaos engineering tool designed to stress-test and validate the robustness of the K3s cluster management system. It performs various failure scenarios, edge cases, and stress conditions to ensure the system behaves correctly under adverse conditions.

## Features

### 🔧 **Test Categories**

1. **Basic Functionality Tests**
   - Version and help commands
   - Command-line interface validation

2. **Cluster Operations Tests**
   - Create multiple clusters
   - List and info operations
   - Invalid cluster operations

3. **Plugin Operations Tests**
   - Install all available plugins
   - Check plugin status
   - Plugin dependency validation

4. **Idempotency Tests**
   - Install plugins multiple times
   - Verify no side effects from repeated operations

5. **Dependency Tests**
   - Test plugin dependency chains
   - Install plugins without dependencies
   - Validate proper error handling

6. **Uninstallation Tests**
   - Remove plugins
   - Handle non-existent plugin removal

7. **Invalid Operations Tests**
   - Non-existent plugins
   - Invalid cluster names
   - Malformed commands

8. **Concurrent Operations Tests**
   - Multiple cluster creation
   - Race condition detection

9. **Resource Exhaustion Tests**
   - Invalid resource specifications
   - Edge case resource values

10. **Operations on Deleted Clusters**
    - Commands against removed clusters
    - Graceful error handling

11. **Edge Cases**
    - Special characters
    - Interrupt handling
    - Command variations

12. **Stress Tests**
    - Rapid install/uninstall cycles
    - System resilience validation

## Usage

### Prerequisites

- Multipass installed and running
- Go development environment
- Make build system
- Sufficient system resources for multiple VMs

### Running the Tests

```bash
# Make the script executable (if not already)
chmod +x chaos.sh

# Run the complete chaos test suite
./chaos.sh
```

### Test Configuration

The script uses the following configurable parameters:

```bash
# Test clusters created during testing
TEST_CLUSTERS=("chaos-test-1" "chaos-test-2" "chaos-test-3")

# Plugins to test
PLUGINS=("cert-manager" "tls" "load-balancer" "nginx-ingress" "argocd" "ingress")

# Invalid plugins for negative testing
INVALID_PLUGINS=("non-existent-plugin" "fake-plugin" "missing-dep")
```

### Output and Logging

- **Colored Console Output**: Real-time test progress with status indicators
- **Log File**: Timestamped log file `chaos-test-YYYYMMDD-HHMMSS.log`
- **Statistics**: Comprehensive test results summary

### Test Result Indicators

- ✅ **Success**: Test passed as expected
- ❌ **Failure**: Test failed unexpectedly
- ⚠️ **Expected Failure**: Test failed as intended (negative test)
- ℹ️ **Info**: Informational messages
- 📊 **Statistics**: Test metrics and summary

## Test Scenarios

### Positive Tests
- Cluster creation and management
- Plugin installation and configuration
- Status checking and listing
- Proper cleanup operations

### Negative Tests
- Invalid cluster names
- Non-existent resources
- Missing dependencies
- Resource exhaustion
- Operations on deleted clusters

### Stress Tests
- Concurrent operations
- Rapid install/uninstall cycles
- Multiple cluster management
- System resource limits

### Edge Cases
- Empty parameters
- Special characters
- Interrupt handling
- Command variations

## Expected Behavior

### What Should Succeed
- Valid cluster operations
- Plugin installations with dependencies met
- Status checks on existing resources
- Proper error messages for invalid operations

### What Should Fail (Gracefully)
- Operations on non-existent clusters
- Invalid plugin installations
- Resource exhaustion scenarios
- Malformed commands

### Idempotency Requirements
- Multiple plugin installations should not cause errors
- Repeated operations should be safe
- System state should remain consistent

## Troubleshooting

### Common Issues

1. **Resource Exhaustion**
   ```bash
   # If you encounter resource issues, reduce concurrent operations
   # or increase system resources
   ```

2. **Multipass Issues**
   ```bash
   # Ensure Multipass is running
   multipass version
   
   # Check available resources
   multipass info
   ```

3. **Network Issues**
   ```bash
   # Check internet connectivity for plugin downloads
   curl -I https://github.com
   ```

### Cleanup

The script includes automatic cleanup mechanisms:

- **Automatic**: Cleanup on script exit (trap)
- **Manual**: Run cleanup section only
- **Force**: Emergency cleanup of all test resources

```bash
# Manual cleanup if needed
./playground cluster list
./playground cluster delete <cluster-name> --force
```

## Integration with CI/CD

The chaos script is designed for integration with continuous integration systems:

```bash
# CI/CD usage
./chaos.sh && echo "Chaos tests passed" || echo "Chaos tests failed"
```

Exit codes:
- `0`: All tests passed (including expected failures)
- `1`: Unexpected test failures occurred

## Performance Considerations

### Resource Usage
- Each test cluster uses VM resources
- Plan for 2-4 GB RAM per cluster
- Allow 5-10 minutes per cluster creation

### Timing
- Full test suite: 30-60 minutes
- Quick validation: 10-15 minutes (reduced clusters)

### Optimization
- Run with adequate system resources
- Consider running subsets for faster feedback
- Use SSD storage for better VM performance

## Customization

### Adding New Tests

```bash
test_custom_scenario() {
    section "Testing Custom Scenario"
    
    run_test "Custom test name" "your-command-here"
    run_test "Expected failure test" "invalid-command" true
}
```

### Modifying Test Parameters

```bash
# Reduce test clusters for faster execution
TEST_CLUSTERS=("chaos-test-1")

# Test specific plugins only
PLUGINS=("cert-manager" "tls")
```

## Security Considerations

- Tests create temporary clusters with default security
- Cleanup ensures no persistent test resources
- No sensitive data stored in test environments
- All operations performed locally via Multipass

## Contributing

When adding new chaos tests:

1. Follow the existing naming convention
2. Include both positive and negative test cases
3. Add proper cleanup for any resources created
4. Update this documentation with new test scenarios
5. Test the new scenarios thoroughly

## Best Practices

1. **Always run cleanup** after testing
2. **Monitor system resources** during execution
3. **Review logs** for unexpected behaviors
4. **Run regularly** to catch regressions
5. **Update tests** when adding new features

## Conclusion

The chaos testing script provides comprehensive validation of system robustness and helps ensure production readiness by testing various failure scenarios and edge cases. Regular execution helps maintain system reliability and catch issues before they reach production environments. 