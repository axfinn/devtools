# GitHub Actions Workflows Documentation

This directory contains all the CI/CD workflows for the DevTools project.

## Workflows Overview

### 1. CI/CD Pipeline (`ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Manual workflow dispatch

**Jobs:**
- **Frontend CI**: Installs dependencies, runs linter and tests, builds the Vue.js app
- **Backend CI**: Runs Go vet, fmt check, tests with coverage, and builds the binary
- **Docker Build**: Builds and optionally pushes Docker images (only on main branch)
- **Security Scan**: Runs Trivy vulnerability scanner on the codebase
- **Quality Gate**: Ensures all checks pass before allowing merge

**Artifacts:**
- Frontend build (dist folder)
- Backend binary
- Coverage reports

### 2. Deployment (`deploy.yml`)

**Triggers:**
- Manual workflow dispatch with environment selection

**Environments:**
- Staging
- Production

**Features:**
- SSH deployment to remote servers
- Docker Compose orchestration
- Health check validation
- Slack notifications (optional)

**Required Secrets:**
- `SSH_PRIVATE_KEY`: SSH private key for server access
- `SERVER_HOST`: Target server hostname
- `SERVER_USER`: SSH user
- `SERVER_PATH`: Path to project on server (default: `/opt/devtools`)
- `SLACK_WEBHOOK_URL`: (Optional) Slack webhook for notifications

### 3. Release (`release.yml`)

**Triggers:**
- Push of version tags (e.g., `v1.0.0`, `v2.1.3`)

**Features:**
- Builds binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates GitHub releases with:
  - Auto-generated changelog
  - Binary attachments
  - Installation instructions
- Builds and pushes versioned Docker images

**Required Secrets:**
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub password/token

**Usage:**
```bash
# Create and push a new version tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### 4. PR Labeler (`pr-labeler.yml`)

**Triggers:**
- Pull request opened, synchronized, or reopened

**Features:**
- Auto-labels PRs based on changed files (frontend, backend, docker, etc.)
- Adds size labels (xs, s, m, l, xl) based on lines changed
- Helps with PR organization and review prioritization

**Configuration:**
- See `.github/labeler.yml` for file path to label mappings

### 5. Stale Management (`stale.yml`)

**Triggers:**
- Daily at midnight (UTC)
- Manual workflow dispatch

**Features:**
- Marks issues stale after 60 days of inactivity
- Marks PRs stale after 30 days of inactivity
- Auto-closes stale issues after 7 days
- Auto-closes stale PRs after 14 days
- Respects exempt labels (pinned, security, bug, etc.)

## Dependabot Configuration

**File:** `.github/dependabot.yml`

**Features:**
- Weekly dependency updates for:
  - Frontend npm packages
  - Backend Go modules
  - GitHub Actions
  - Docker base images
- Automatic PR creation with proper labels
- Configurable update limits and schedules

## Setup Instructions

### 1. Required GitHub Secrets

Navigate to **Settings > Secrets and variables > Actions** and add:

#### For Docker builds:
- `DOCKER_USERNAME`: Your Docker Hub username
- `DOCKER_PASSWORD`: Your Docker Hub access token

#### For deployments:
- `SSH_PRIVATE_KEY`: Private SSH key for server access
- `SERVER_HOST`: Server hostname or IP
- `SERVER_USER`: SSH username
- `SERVER_PATH`: Deployment path on server

#### For notifications (optional):
- `SLACK_WEBHOOK_URL`: Slack incoming webhook URL

### 2. Branch Protection Rules

Recommended settings for `main` branch:

1. Go to **Settings > Branches > Branch protection rules**
2. Add rule for `main`
3. Enable:
   - Require a pull request before merging
   - Require status checks to pass before merging
     - Select: `Frontend CI`, `Backend CI`, `Quality Gate`
   - Require branches to be up to date before merging
   - Require conversation resolution before merging

### 3. GitHub Environments

Create environments for deployments:

1. Go to **Settings > Environments**
2. Create `staging` and `production` environments
3. Add protection rules:
   - Required reviewers for production
   - Deployment branches (limit to `main` for production)

### 4. Customize Workflows

#### Update Dependabot reviewers:
Edit `.github/dependabot.yml` and replace `your-github-username` with your actual GitHub username.

#### Adjust test commands:
Once you add tests, update these lines in `ci.yml`:
```yaml
# Frontend
- name: Run tests
  run: npm test  # Remove || echo "..." part

# Backend
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out ./...  # Remove || echo "..." part
```

#### Configure Docker image names:
Update the Docker image name in workflows if not using `${{ secrets.DOCKER_USERNAME }}/devtools`:
- `ci.yml`: Line with `images:` in metadata step
- `release.yml`: Line with `tags:` in build-push step

## Workflow Status Badges

Add these badges to your README.md:

```markdown
![CI/CD Pipeline](https://github.com/YOUR_USERNAME/devtools/actions/workflows/ci.yml/badge.svg)
![Security Scan](https://github.com/YOUR_USERNAME/devtools/actions/workflows/ci.yml/badge.svg?event=schedule)
```

## Best Practices

1. **Always run tests locally before pushing**
   ```bash
   # Frontend
   cd frontend && npm test && npm run build

   # Backend
   cd backend && go test ./... && go build
   ```

2. **Create feature branches**
   ```bash
   git checkout -b feature/new-feature
   ```

3. **Use semantic commit messages**
   ```
   feat: add new JSON tool feature
   fix: resolve diff comparison bug
   chore: update dependencies
   docs: improve API documentation
   ```

4. **Tag releases properly**
   ```bash
   # Format: vMAJOR.MINOR.PATCH
   git tag -a v1.2.3 -m "Release version 1.2.3"
   ```

5. **Monitor workflow runs**
   - Check the Actions tab regularly
   - Fix failing builds promptly
   - Review security scan results

## Troubleshooting

### Docker push fails
- Verify `DOCKER_USERNAME` and `DOCKER_PASSWORD` secrets are set correctly
- Ensure Docker Hub token has write permissions
- Check if repository name matches in workflow files

### Deployment fails
- Verify SSH key is correct and has proper permissions
- Ensure server is accessible from GitHub Actions IP ranges
- Check server has Docker and docker-compose installed
- Verify `SERVER_PATH` exists on the target server

### Tests not running
- Ensure test scripts are defined in `package.json` (frontend)
- Verify Go test files exist with `_test.go` suffix (backend)
- Remove `continue-on-error: true` once tests are implemented

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
- [Dependabot Configuration](https://docs.github.com/en/code-security/dependabot)
- [Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches)
