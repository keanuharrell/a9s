# a9s 🚀

**The k9s for AWS** - An interactive Terminal UI for AWS infrastructure management written in Go.

## What is a9s?

a9s brings the intuitive k9s experience to AWS! Navigate your AWS infrastructure with an interactive Terminal UI that makes managing EC2, IAM, and S3 resources as smooth as managing Kubernetes with k9s.

**✨ Key Features:**
- 🖥️ **Interactive TUI** - Real-time, navigable interface (just run `a9s`)
- ⚡ **CLI Mode** - Traditional command-line interface (`a9s ec2 list`, etc.)
- 🔄 **Auto-refresh** - Live updates every 5 seconds
- 🎯 **Multi-service** - EC2, IAM, S3 in one tool
- 🎨 **Beautiful UI** - Powered by BubbleTea with colors and tables

## Features

- **EC2 Management**: List and monitor EC2 instances across regions
- **IAM Security Auditing**: Identify high-risk roles and permissions
- **S3 Bucket Cleanup**: Find and remove empty or untagged buckets
- **Multi-format Output**: JSON or table format for all commands
- **AWS Profile Support**: Work with multiple AWS accounts seamlessly

## Installation

### Homebrew (Recommended for macOS)

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

### Using Docker

```bash
docker build -t a9s .
docker run --rm -v ~/.aws:/root/.aws a9s --help
```

### Package Managers

```bash
# Snap (Linux)
snap install a9s

# Go Install
go install github.com/keanuharrell/a9s@latest
```

## Usage

### 🖥️ Interactive TUI Mode (Recommended)

Launch the beautiful Terminal UI:
```bash
a9s                    # Launch TUI with default profile
a9s --profile prod     # Use specific AWS profile  
a9s --region eu-west-1 # Use specific region
```

**TUI Navigation:**
- `[1,2,3]` - Switch between EC2, IAM, S3 views
- `[↑↓]` - Navigate items in current view  
- `[Enter]` - View details
- `[r]` - Refresh current view
- `[s/t]` - Start/Stop EC2 instances (demo)
- `[q]` - Quit
- `[?]` - Help

### ⚡ CLI Mode

#### EC2 Commands

List all EC2 instances:
```bash
a9s ec2 list
a9s ec2 list --region us-west-2
a9s ec2 list --profile production --output json
```

#### IAM Commands

Audit IAM roles for security risks:
```bash
a9s iam audit
a9s iam audit --output json
a9s iam audit --profile security-audit
```

### S3 Commands

Analyze and cleanup S3 buckets:
```bash
a9s s3 cleanup --dry-run
a9s s3 cleanup --output json
a9s s3 cleanup  # Actually delete buckets (use with caution!)
```

## Global Flags

- `--profile`: AWS profile to use (from ~/.aws/credentials)
- `--region`: AWS region (overrides default region)
- `--output`: Output format (json|table) - default: table
- `--dry-run`: Simulate actions without making changes
- `--config`: Path to configuration file (optional)

## Configuration

a9s uses standard AWS credentials. Configure your credentials using:

```bash
aws configure
# or
export AWS_PROFILE=your-profile
export AWS_REGION=us-east-1
```

## Examples

### Find all running EC2 instances in JSON format
```bash
a9s ec2 list --output json | jq '.[] | select(.state == "running")'
```

### Audit IAM roles and save results
```bash
a9s iam audit --output json > iam-audit-report.json
```

### Preview S3 cleanup without deleting
```bash
a9s s3 cleanup --dry-run
```

## Building

### Requirements
- Go 1.21 or higher
- AWS credentials configured

### Build for current platform
```bash
go build -o a9s .
```

### Cross-compile for different platforms
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o a9s-linux-amd64

# macOS
GOOS=darwin GOARCH=amd64 go build -o a9s-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o a9s-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o a9s-windows-amd64.exe
```

## Docker

Build the Docker image:
```bash
docker build -t a9s .
```

Run with AWS credentials:
```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  -e AWS_PROFILE=default \
  a9s ec2 list
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Security

- Never commit AWS credentials
- Use IAM roles when running on EC2
- Always use `--dry-run` before destructive operations
- Review IAM audit results regularly

## License

MIT License - see LICENSE file for details

## Author

Keanu Harrell

## Roadmap

- [ ] Support for more AWS services (RDS, Lambda, etc.)
- [ ] Configuration file support for default settings
- [ ] Interactive mode for complex operations
- [ ] Cost analysis features
- [ ] Multi-cloud support (Azure, GCP)
- [ ] Automated remediation for common issues
