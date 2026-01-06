# a9s

**The k9s for AWS** - An interactive Terminal UI for AWS infrastructure management.

![Go Version](https://img.shields.io/github/go-mod/go-version/keanuharrell/a9s)
![License](https://img.shields.io/github/license/keanuharrell/a9s)
![Release](https://img.shields.io/github/v/release/keanuharrell/a9s)

## Overview

a9s brings the intuitive [k9s](https://k9scli.io/) experience to AWS. Navigate your AWS infrastructure with a beautiful Terminal UI that makes managing cloud resources simple and efficient.

## Features

- **Interactive TUI** - Real-time, keyboard-driven interface
- **Multi-Service Support** - EC2, IAM, S3, Lambda in one tool
- **Profile & Region Switching** - Switch AWS profiles and regions on the fly
- **Auto-refresh** - Live updates for resource status
- **Keyboard-First** - Navigate entirely with keyboard shortcuts

### Supported Services

| Service | Features |
|---------|----------|
| **EC2** | List instances, start/stop/reboot, view status |
| **IAM** | List roles, security analysis, permission auditing |
| **S3** | List buckets, analyze storage, delete empty buckets |
| **Lambda** | List functions, view configuration, invoke functions |

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap keanuharrell/a9s
brew install a9s
```

### From Source

```bash
git clone https://github.com/keanuharrell/a9s.git
cd a9s
go build -o a9s .
sudo mv a9s /usr/local/bin/
```

### Go Install

```bash
go install github.com/keanuharrell/a9s@latest
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/keanuharrell/a9s/releases).

## Usage

```bash
# Launch TUI with default profile
a9s

# Use specific AWS profile
a9s --profile production

# Use specific region
a9s --region eu-west-1

# Combine options
a9s --profile prod --region us-east-1
```

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| `1` | Switch to EC2 view |
| `2` | Switch to IAM view |
| `3` | Switch to S3 view |
| `4` | Switch to Lambda view |
| `p` | Change AWS profile |
| `R` | Change AWS region |
| `r` | Refresh current view |
| `q` / `Ctrl+C` | Quit |

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select / View details |

### Service-Specific

**EC2:**
| Key | Action |
|-----|--------|
| `s` | Start instance |
| `t` | Stop instance |
| `b` | Reboot instance |

**S3:**
| Key | Action |
|-----|--------|
| `a` | Analyze bucket |
| `d` | Delete bucket |

**Lambda:**
| Key | Action |
|-----|--------|
| `i` | Invoke function |

## Configuration

a9s uses standard AWS credentials from `~/.aws/credentials` and `~/.aws/config`.

```bash
# Configure AWS credentials
aws configure

# Or set environment variables
export AWS_PROFILE=your-profile
export AWS_REGION=us-east-1
```

### Optional Config File

Create `~/.config/a9s/config.yaml`:

```yaml
defaults:
  profile: default
  region: us-east-1

services:
  - ec2
  - iam
  - s3
  - lambda
```

## Requirements

- AWS credentials configured
- Go 1.21+ (for building from source)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features (triggers minor version bump)
   - `fix:` for bug fixes (triggers patch version bump)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Security

- Never commit AWS credentials
- Use IAM roles with least privilege
- Use `--dry-run` before destructive operations
- Report security issues via GitHub Security Advisories

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

[Keanu Harrell](https://github.com/keanuharrell)
