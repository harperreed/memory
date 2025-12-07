# Release Process

## Prerequisites

1. Install goreleaser:
```bash
go install github.com/goreleaser/goreleaser@latest
```

2. Set up GitHub token:
```bash
export GITHUB_TOKEN="your-github-token"
```

3. (Optional) Set up Homebrew tap token for brew formula:
```bash
export HOMEBREW_TAP_TOKEN="your-homebrew-tap-token"
```

## Testing a Release Locally

Test the release configuration without creating a release:

```bash
make release-test
```

Create a snapshot release (local build only, not published):

```bash
make release-snapshot
```

## Creating a Release

### 1. Update Version

Ensure all changes are committed and pushed to main.

### 2. Create and Push Tag

```bash
# Create an annotated tag
git tag -a v0.1.0 -m "Release v0.1.0"

# Push the tag
git push origin v0.1.0
```

### 3. GitHub Actions Automation

Once the tag is pushed, GitHub Actions will automatically:
- Run all tests
- Build binaries for macOS (amd64 and arm64)
- Create a GitHub release with:
  - Changelog
  - Binary archives
  - Checksums
- (If HOMEBREW_TAP_TOKEN is set) Update Homebrew formula

### 4. Verify Release

1. Check GitHub Actions: https://github.com/harperreed/remember-standalone/actions
2. Verify release page: https://github.com/harperreed/remember-standalone/releases
3. Test installation:
   ```bash
   # Download and test
   curl -L https://github.com/harperreed/remember-standalone/releases/download/v0.1.0/memory_0.1.0_Darwin_x86_64.tar.gz | tar xz
   ./memory version
   ```

## Versioning

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes

Examples:
- `v0.1.0` - Initial release
- `v0.2.0` - Add new CLI commands
- `v0.2.1` - Fix bug in search command
- `v1.0.0` - First stable release

## Changelog

Changelog is automatically generated from commit messages. Use conventional commits:

- `feat:` - New features
- `fix:` - Bug fixes
- `perf:` - Performance improvements
- `docs:` - Documentation changes
- `chore:` - Maintenance tasks
- `test:` - Test updates

Example:
```bash
git commit -m "feat: add vector search to CLI"
git commit -m "fix: handle empty search results gracefully"
```

## Rollback

If a release has issues:

1. Delete the tag locally and remotely:
   ```bash
   git tag -d v0.1.0
   git push origin :refs/tags/v0.1.0
   ```

2. Delete the GitHub release from the web interface

3. Fix the issues and create a new tag

## Platform Support

Currently supported platforms:
- macOS (amd64, arm64)

Linux support requires Docker-based builds or platform-specific compilation due to CGO requirements for SQLite. Users can build from source on Linux.

## Homebrew Formula

If HOMEBREW_TAP_TOKEN is configured, releases automatically update the Homebrew formula at:
https://github.com/harperreed/homebrew-tap

Users can then install with:
```bash
brew tap harperreed/tap
brew install memory
```
