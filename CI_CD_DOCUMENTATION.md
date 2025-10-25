# CI/CD Pipeline Documentation

## Overview

This project uses GitHub Actions for comprehensive CI/CD automation including continuous integration, security scanning, and automated releases.

---

## Table of Contents

- [Workflows](#workflows)
  - [CI Workflow](#ci-workflow)
  - [Security Workflow](#security-workflow)
  - [Release Workflow](#release-workflow)
- [Setup Instructions](#setup-instructions)
- [Branch Protection](#branch-protection)
- [Secrets Configuration](#secrets-configuration)
- [Usage Examples](#usage-examples)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

---

## Workflows

### CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Manual workflow dispatch

**Jobs:**

#### 1. Lint & Format Check
- **Purpose**: Ensure code quality and consistency
- **Tools**:
  - golangci-lint (comprehensive linting)
  - gofmt (formatting check)
  - go vet (static analysis)
  - go mod tidy verification
- **Duration**: ~5-10 minutes

#### 2. Test Matrix
- **Purpose**: Run unit tests across multiple platforms
- **Matrix**:
  - OS: Ubuntu, macOS, Windows
  - Go Version: 1.23
- **Features**:
  - Race condition detection (`-race`)
  - Code coverage report
  - Codecov integration
- **Duration**: ~10-15 minutes per platform

#### 3. Benchmarks
- **Purpose**: Performance regression detection
- **Features**:
  - Run all benchmarks with memory profiling
  - Store results as artifacts
  - Compare against previous runs (PR only)
  - Alert on >150% performance degradation
- **Duration**: ~10-15 minutes

#### 4. Build Verification
- **Purpose**: Verify application builds successfully
- **Matrix**: Same as tests
- **Features**:
  - Cross-platform builds
  - Optimized binaries (`-ldflags="-s -w"`)
  - Upload artifacts
- **Duration**: ~5-10 minutes per platform

#### 5. Integration Tests
- **Purpose**: End-to-end testing (if applicable)
- **Features**:
  - Runs after unit tests pass
  - Optional (skipped if no integration tests found)
- **Duration**: ~15-20 minutes

#### 6. Code Coverage
- **Purpose**: Track test coverage metrics
- **Features**:
  - Generate HTML coverage report
  - Calculate coverage percentage
  - Comment on PRs with coverage stats
  - Upload artifacts
- **Duration**: ~5 minutes

---

### Security Workflow (`.github/workflows/security.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Daily scheduled scan (2 AM UTC)
- Manual workflow dispatch

**Jobs:**

#### 1. GoSec Security Scanner
- **Purpose**: Go-specific security vulnerability detection
- **Output**: SARIF format uploaded to Security tab
- **Checks**: SQL injection, hardcoded credentials, weak crypto, etc.

#### 2. Go Vulnerability Check (govulncheck)
- **Purpose**: Check for known vulnerabilities in dependencies
- **Database**: Official Go vulnerability database
- **Action**: Fails on known vulnerabilities

#### 3. Nancy Dependency Scanner
- **Purpose**: OSS Index vulnerability scanning
- **Scope**: All dependencies
- **Action**: Continue on error (informational)

#### 4. Trivy Security Scanner
- **Purpose**: Comprehensive security scanning
- **Scope**: Filesystem, dependencies, misconfigurations
- **Severity**: CRITICAL, HIGH, MEDIUM
- **Output**: SARIF format

#### 5. CodeQL Analysis
- **Purpose**: Advanced semantic code analysis
- **Language**: Go
- **Queries**: Security and quality rules
- **Output**: Detailed findings in Security tab

#### 6. Gitleaks Secret Scanner
- **Purpose**: Detect hardcoded secrets and credentials
- **Scope**: Full git history
- **Action**: Fails on secrets found

#### 7. License Compliance
- **Purpose**: Ensure dependency licenses are compliant
- **Tool**: go-licenses
- **Checks**: Forbidden/restricted license types
- **Output**: License report (CSV)

---

### Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Push tags matching `v*.*.*` (e.g., v1.0.0)
- Manual workflow dispatch with version input

**Jobs:**

#### 1. Create Release
- **Purpose**: Generate GitHub release with changelog
- **Features**:
  - Automatic changelog generation
  - Draft/prerelease detection
  - Release notes template

#### 2. Build Multi-Platform Binaries
- **Matrix**:
  - Linux: amd64, arm64
  - macOS: amd64, arm64
  - Windows: amd64
- **Features**:
  - Optimized builds
  - Version injection
  - Archive creation (tar.gz/zip)
  - Upload to release

#### 3. Generate Checksums
- **Purpose**: Provide SHA256 checksums for verification
- **Output**: `checksums.txt` file

#### 4. Build & Push Docker Image
- **Purpose**: Create multi-arch Docker images
- **Registry**: GitHub Container Registry (ghcr.io)
- **Tags**:
  - Semver: `v1.0.0`, `v1.0`, `v1`
  - Latest (for main branch)
- **Platforms**: linux/amd64, linux/arm64

#### 5. Post-Release Tasks
- **Purpose**: Update `latest` tag, notifications
- **Features**:
  - Tag update
  - Release notification

---

## Setup Instructions

### 1. Initial Setup

```bash
# Clone repository
git clone <repository-url>
cd stream

# Verify workflows
ls -la .github/workflows/
```

### 2. Configure GitHub Repository

#### Enable Workflows
1. Go to repository **Settings** → **Actions** → **General**
2. Under "Actions permissions", select **Allow all actions and reusable workflows**
3. Under "Workflow permissions", select **Read and write permissions**
4. Check **Allow GitHub Actions to create and approve pull requests**

#### Enable Security Features
1. Go to **Security** → **Code scanning**
2. Enable **CodeQL analysis** (if not auto-enabled)
3. Go to **Security** → **Secret scanning**
4. Enable **Secret scanning** and **Push protection**

#### Enable Dependabot
1. Go to **Security** → **Dependabot**
2. Enable **Dependabot alerts**
3. Enable **Dependabot security updates**
4. Dependabot version updates are configured via `.github/dependabot.yml`

### 3. Configure Secrets

Go to repository **Settings** → **Secrets and variables** → **Actions**

**Required Secrets:**
```bash
# Optional: For Codecov integration
CODECOV_TOKEN=<your-codecov-token>

# Optional: For Gitleaks (free tier)
GITLEAKS_LICENSE=<your-license-key>
```

**How to get tokens:**
- **CODECOV_TOKEN**: Sign up at https://codecov.io and link repository
- **GITLEAKS_LICENSE**: Free at https://gitleaks.io

### 4. Configure Branch Protection

Go to repository **Settings** → **Branches** → **Add rule**

**Branch name pattern**: `main`

**Protection rules:**
- ✅ Require pull request reviews before merging (1 approver)
- ✅ Dismiss stale pull request approvals when new commits are pushed
- ✅ Require status checks to pass before merging
  - Select: `CI Success`, `Security Success`
- ✅ Require branches to be up to date before merging
- ✅ Require conversation resolution before merging
- ✅ Require linear history
- ✅ Include administrators

---

## Secrets Configuration

### GitHub Secrets Overview

| Secret Name | Required | Purpose | How to Obtain |
|-------------|----------|---------|---------------|
| `CODECOV_TOKEN` | Optional | Code coverage reporting | https://codecov.io |
| `GITLEAKS_LICENSE` | Optional | Enhanced secret scanning | https://gitleaks.io |
| `GITHUB_TOKEN` | Auto | GitHub API access | Auto-generated |

### Adding Secrets

```bash
# Using GitHub CLI
gh secret set CODECOV_TOKEN

# Or via GitHub UI:
# Settings → Secrets and variables → Actions → New repository secret
```

---

## Usage Examples

### Running CI on Pull Request

```bash
# Create feature branch
git checkout -b feature/new-operator

# Make changes
# ... code changes ...

# Commit and push
git add .
git commit -m "feat: add new operator"
git push origin feature/new-operator

# Create PR (triggers CI automatically)
gh pr create --title "Add new operator" --body "Description"
```

### Creating a Release

```bash
# Method 1: Via Git Tag (recommended)
git tag v1.0.0
git push origin v1.0.0
# Release workflow triggers automatically

# Method 2: Via GitHub UI
# Go to Releases → Draft a new release → Create tag → Publish

# Method 3: Via GitHub CLI
gh release create v1.0.0 --generate-notes

# Method 4: Manual workflow dispatch
# Actions → Release → Run workflow → Enter version
```

### Manual Workflow Trigger

```bash
# Using GitHub CLI
gh workflow run ci.yml

gh workflow run security.yml

gh workflow run release.yml -f version=v1.0.0

# Or via GitHub UI:
# Actions → Select workflow → Run workflow
```

### Viewing Workflow Results

```bash
# List recent workflow runs
gh run list

# View specific run details
gh run view <run-id>

# View run logs
gh run view <run-id> --log

# Download artifacts
gh run download <run-id>
```

---

## Troubleshooting

### Common Issues

#### 1. golangci-lint Timeout

**Error**: `golangci-lint: timeout exceeded`

**Solution**:
```yaml
# In .github/workflows/ci.yml, increase timeout:
args: --timeout=10m --config=.golangci.yml
```

#### 2. Test Failures on Specific OS

**Issue**: Tests pass locally but fail on CI

**Debug**:
```bash
# Run tests with verbose output locally
go test -v -race ./...

# Check for OS-specific code
grep -r "runtime.GOOS" .

# Review test logs in GitHub Actions
```

#### 3. Docker Build Failures

**Error**: `failed to solve: failed to resolve source metadata`

**Solution**:
```bash
# Test Docker build locally
docker build -t stream:test .

# Check Dockerfile syntax
docker build --no-cache -t stream:test .

# Verify .dockerignore
cat .dockerignore
```

#### 4. Codecov Upload Failures

**Error**: `Error uploading coverage reports`

**Solution**:
```bash
# Verify CODECOV_TOKEN is set
# Settings → Secrets → Check CODECOV_TOKEN

# Try without fail_ci_if_error
fail_ci_if_error: false
```

#### 5. Dependency Download Failures

**Error**: `go: downloading ... failed`

**Solution**:
```yaml
# Add retry mechanism
- name: Download dependencies
  run: |
    for i in {1..3}; do
      go mod download && break || sleep 15
    done
```

---

## Best Practices

### 1. Commit Message Convention

Follow Conventional Commits specification:

```bash
# Format
<type>(<scope>): <subject>

# Types
feat:     New feature
fix:      Bug fix
docs:     Documentation changes
style:    Code style changes (formatting)
refactor: Code refactoring
perf:     Performance improvements
test:     Test additions/changes
chore:    Build process, dependencies
ci:       CI configuration changes

# Examples
feat(operators): add processSurveyAnswer operator
fix(api): handle nil pointer in request handler
docs(readme): update installation instructions
perf(cache): optimize buffer allocation
```

### 2. Pull Request Guidelines

**Before creating PR:**
```bash
# Run tests locally
go test ./...

# Run linter
golangci-lint run

# Check formatting
gofmt -l .

# Run benchmarks (if applicable)
go test -bench=. ./...
```

**PR Description Template:**
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Benchmarks run (if performance-related)

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings generated
```

### 3. Release Management

**Versioning (Semantic Versioning):**
- **MAJOR**: Incompatible API changes (v2.0.0)
- **MINOR**: New functionality, backward-compatible (v1.1.0)
- **PATCH**: Bug fixes, backward-compatible (v1.0.1)
- **Pre-release**: Alpha, beta, rc (v1.0.0-alpha.1)

**Release Checklist:**
```bash
# 1. Update CHANGELOG.md
# 2. Update version in code (if applicable)
# 3. Commit changes
git add CHANGELOG.md
git commit -m "chore: prepare v1.0.0 release"

# 4. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 5. Verify release workflow
gh run list --workflow=release.yml

# 6. Test released binaries
gh release download v1.0.0
```

### 4. Security Best Practices

**Code Review:**
- Review all dependency updates from Dependabot
- Check security scan results before merging
- Address CRITICAL and HIGH severity findings immediately

**Secrets Management:**
- Never commit secrets to repository
- Use GitHub Secrets for sensitive data
- Rotate secrets regularly
- Enable secret scanning and push protection

**Dependency Management:**
- Keep dependencies up to date
- Review license compatibility
- Pin versions for reproducible builds
- Monitor vulnerability alerts

### 5. Performance Monitoring

**Benchmark Tracking:**
```bash
# Run benchmarks locally
go test -bench=. -benchmem ./...

# Compare with previous results
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt

# Track memory allocations
go test -bench=. -benchmem -memprofile=mem.prof ./...
go tool pprof mem.prof
```

**Coverage Goals:**
- Overall: >80%
- Critical paths: >90%
- New code: 100%

---

## Workflow Badges

Add these badges to your README.md:

```markdown
![CI](https://github.com/USERNAME/REPO/actions/workflows/ci.yml/badge.svg)
![Security](https://github.com/USERNAME/REPO/actions/workflows/security.yml/badge.svg)
![Release](https://github.com/USERNAME/REPO/actions/workflows/release.yml/badge.svg)
[![codecov](https://codecov.io/gh/USERNAME/REPO/branch/main/graph/badge.svg)](https://codecov.io/gh/USERNAME/REPO)
[![Go Report Card](https://goreportcard.com/badge/github.com/USERNAME/REPO)](https://goreportcard.com/report/github.com/USERNAME/REPO)
```

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint](https://golangci-lint.run/)
- [Go Testing](https://pkg.go.dev/testing)
- [Codecov](https://docs.codecov.com/)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [CodeQL](https://codeql.github.com/)
- [Trivy](https://github.com/aquasecurity/trivy)

---

## Contact & Support

For questions or issues related to CI/CD:
1. Check this documentation
2. Review workflow logs in GitHub Actions
3. Open an issue on GitHub
4. Contact repository maintainers

---

**Last Updated:** 2025-10-26
**Version:** 1.0.0
