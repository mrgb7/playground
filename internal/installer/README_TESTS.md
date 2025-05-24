# ArgoCD Installer Integration Tests

This document describes the comprehensive test suite for the ArgoCD installer functionality.

## Test Overview

The integration tests are located in `argo_test.go` and provide comprehensive coverage of the ArgoCD installer functionality, including unit tests, error handling, and edge cases.

## Test Categories

### 1. Constructor Tests (`TestNewArgoInstaller`)

Tests the `NewArgoInstaller` constructor function:

- **Invalid Config**: Tests behavior with malformed kubeconfig
- **Empty Config**: Tests behavior with empty kubeconfig string
- **Empty Cluster Name**: Tests behavior with empty cluster name (should still work)

**Coverage**: Constructor validation, error handling, default value assignment

### 2. Connection Validation Tests (`TestArgoInstaller_ValidateArgoConnection`)

Tests the ArgoCD connection validation functionality:

- **Valid Connection**: Tests with properly formatted server address
- **No Connection**: Tests with empty server address (should fail)
- **Invalid Address Format**: Tests with malformed address (currently passes, may need enhancement)

**Coverage**: Connection state validation, error detection

### 3. Port Forward Management Tests

#### `TestArgoInstaller_closePortForward`
Tests proper cleanup of active port forward connections:
- Verifies context cancellation
- Ensures cleanup of cancel function

#### `TestArgoInstaller_closePortForward_NilCancel`
Tests safe handling when no active port forward exists:
- Verifies no panic with nil cancel function

**Coverage**: Resource cleanup, graceful shutdown

### 4. Install Options Validation Tests

#### `TestInstallOptions_Validation`
Tests basic validation rules:
- **Valid Options**: Complete, valid options object
- **Missing Application Name**: Should fail validation
- **Missing Repo URL**: Should fail validation
- **Nil Options**: Should fail validation
- **Empty Strings**: Should fail validation with empty required fields

#### `TestInstallOptions_ComplexValidation`
Tests complex real-world scenarios:
- **Complete Configuration**: All fields populated with realistic values
- **Minimal Configuration**: Only required fields
- **Special Characters**: Tests handling of special characters in names and URLs

**Coverage**: Input validation, parameter checking

### 5. Struct Initialization Tests (`TestArgoInstaller_StructInitialization`)

Tests proper struct initialization:
- **Default Initialization**: Using default constants
- **Custom Initialization**: Custom values for all fields

**Coverage**: Struct field assignment, value preservation

### 6. Constants Tests (`TestArgoInstaller_Constants`)

Validates the defined constants:
- `DefaultArgoNamespace` = "argocd"
- `DefaultArgoServerPort` = 443
- `DefaultLocalPort` = 8080

**Coverage**: Constant definitions, default values

### 7. Default Values Tests (`TestArgoInstaller_DefaultValues`)

Tests that default values are properly assigned and accessible.

**Coverage**: Default value assignment

### 8. Nil Parameter Handling Tests

#### `TestArgoInstaller_Install_NilOptions`
Tests that Install method properly handles nil options parameter.

#### `TestArgoInstaller_UnInstall_NilOptions`
Tests that UnInstall method properly handles nil options parameter.

**Coverage**: Nil parameter safety, error handling

## Running the Tests

```bash
# Run all installer tests
go test -v ./internal/installer/

# Run specific test
go test -v ./internal/installer/ -run TestNewArgoInstaller

# Run with coverage
go test -cover ./internal/installer/
```

## Test Philosophy

The tests follow these principles:

1. **Unit Testing Focus**: Tests focus on individual method behavior rather than full integration
2. **Error Case Coverage**: Comprehensive testing of error scenarios and edge cases
3. **Nil Safety**: All public methods tested for nil parameter handling
4. **Value Validation**: Verification of proper value assignment and preservation
5. **Resource Cleanup**: Testing of proper resource management and cleanup

## Limitations

The current test suite does not include:

- Full Kubernetes integration (requires running cluster)
- Actual port forwarding (requires network setup)
- Real ArgoCD API calls (requires ArgoCD server)
- Secret retrieval from live cluster

These limitations are intentional to keep tests fast, reliable, and environment-independent.

## Future Enhancements

Potential areas for test expansion:

1. Mock Kubernetes client integration
2. Network connectivity simulation
3. ArgoCD API response mocking
4. Performance benchmarking
5. Concurrent operation testing

## Dependencies

Test dependencies:
- Standard Go testing framework
- Context package for cancellation testing
- Time package for timeout testing

No external testing frameworks or mocking libraries are required. 