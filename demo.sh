#!/bin/bash

# a9s Demo Script
# This script demonstrates the capabilities of a9s

echo "🚀 a9s - The k9s for AWS Demo"
echo "============================"
echo

echo "📋 Available commands:"
echo "1. TUI Mode (Interactive):"
echo "   a9s                    # Launch interactive Terminal UI"
echo "   a9s tui                # Explicit TUI launch"
echo

echo "2. CLI Mode (Traditional):"
echo "   a9s ec2 list           # List EC2 instances"
echo "   a9s iam audit          # Audit IAM roles"  
echo "   a9s s3 cleanup         # S3 cleanup analysis"
echo

echo "3. Global flags:"
echo "   --profile prod         # Use AWS profile"
echo "   --region eu-west-1     # Set AWS region"
echo "   --output json          # JSON output format"
echo "   --dry-run              # Safe mode (no changes)"
echo

echo "💡 Tips:"
echo "- Run 'a9s' (no arguments) for the beautiful TUI experience"
echo "- Use 'a9s --help' for detailed help"
echo "- Configure AWS credentials first: 'aws configure'"
echo

echo "🎯 Example Usage:"
echo "# Interactive TUI (recommended)"
echo "a9s"
echo
echo "# CLI commands"  
echo "a9s ec2 list --region us-east-1 --output json"
echo "a9s iam audit --profile security-team"
echo "a9s s3 cleanup --dry-run"
echo

if command -v a9s &> /dev/null; then
    echo "✅ a9s is installed and ready to use!"
    echo "Run: a9s --version"
    a9s --version
else
    echo "❌ a9s is not installed. Install it with:"
    echo "   brew tap keanuharrell/a9s && brew install a9s"
    echo "   # or build from source: go build -o a9s ."
fi