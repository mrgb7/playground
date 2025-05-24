# Contributing to Playground

Thank you for your interest in contributing to Playground! This document provides guidelines and information for contributors.

## Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/mrgb7/playground.git
   cd playground
   ```

2. **Set up development environment**:
   ```bash
   make dev-setup
   ```

3. **Run tests**:
   ```bash
   make test
   ```

4. **Build the project**:
   ```bash
   make build
   ```

## Semantic Versioning

This project follows [Semantic Versioning (SemVer)](https://semver.org/). Version numbers follow the format `MAJOR.MINOR.PATCH`:

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality in a backwards compatible manner
- **PATCH**: Backwards compatible bug fixes

## Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/) to automatically determine version bumps and generate changelogs. 

### Commit Message Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: A new feature (triggers MINOR version bump)
- **fix**: A bug fix (triggers PATCH version bump)
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to the build process or auxiliary tools

### Breaking Changes

To indicate breaking changes, add `!` after the type or include `BREAKING CHANGE:` in the footer:

```bash
feat!: remove deprecated API endpoint
```

or

```bash
feat: add new authentication system

BREAKING CHANGE: The old authentication system has been removed.
```

This triggers a MAJOR version bump.

### Examples

```bash
# Patch version bump
fix: resolve memory leak in worker pool
fix(auth): handle expired tokens correctly

# Minor version bump  
feat: add user profile management
feat(api): implement rate limiting

# Major version bump
feat!: redesign authentication system
refactor!: change configuration file format
```

## Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards

3. **Run pre-commit checks**:
   ```bash
   make pre-commit
   ```

4. **Commit your changes** using conventional commit format:
   ```bash
   git commit -m "feat: add new awesome feature"
   ```

5. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request** to the `main` branch

## Pull Request Guidelines

- **Title**: Use conventional commit format in PR title
- **Description**: Provide clear description of changes
- **Tests**: Include tests for new functionality
- **Documentation**: Update documentation if needed
- **Breaking Changes**: Clearly document any breaking changes

## Continuous Integration

Our CI pipeline includes:

### Pull Request Checks
- **Tests**: Unit tests with race detection
- **Code Format**: `gofmt` formatting check
- **Linting**: `golangci-lint` analysis
- **Security**: Security vulnerability scanning
- **Build**: Multi-platform build verification

### Release Process
When changes are merged to `main`:

1. **Automatic Version Detection**: Based on conventional commits since last release
2. **Tag Creation**: Semantic version tag is created
3. **Release Build**: Binaries are built for Linux and macOS
4. **GitHub Release**: Release with changelog is created automatically

## Manual Release

You can trigger a manual release with specific version:

1. **Go to Actions tab** in GitHub
2. **Select "Release" workflow**
3. **Click "Run workflow"**
4. **Specify version type**:
   - `auto` (default): Analyze commits automatically
   - `major`: Force major version bump
   - `minor`: Force minor version bump  
   - `patch`: Force patch version bump
   - `v1.2.3`: Specific version number

## Code Quality Standards

- **Go Version**: Go 1.24+
- **Formatting**: Use `gofmt` for formatting
- **Linting**: Pass `golangci-lint` checks
- **Test Coverage**: Maintain test coverage for new code
- **Documentation**: Include godoc comments for public APIs

## Supported Platforms

Release binaries are built for:
- Linux AMD64
- macOS Intel (AMD64)
- macOS Apple Silicon (ARM64)

## Getting Help

- **Issues**: Create an issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Code Review**: Request reviews from maintainers

## Release Assets

Each release includes:
- Source code
- Pre-built binaries for supported platforms
- Checksums
- Auto-generated changelog

### Installation Example

```bash
# Download for Linux
curl -L -o playground.tar.gz https://github.com/mrgb7/playground/releases/latest/download/playground-vX.Y.Z-linux-amd64.tar.gz
tar -xzf playground.tar.gz
chmod +x playground-linux-amd64
sudo mv playground-linux-amd64 /usr/local/bin/playground

# Verify installation
playground version --verbose
``` 