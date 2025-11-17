# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Currently supported versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

The a9s team takes security bugs seriously. We appreciate your efforts to responsibly disclose your findings.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to:

- **Email**: <keanuharrell@icloud.com> (replace with real email)
- **Subject**: [SECURITY] Brief description of the vulnerability

### What to Include

Please include the following information:

1. **Type of issue** (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
2. **Full paths** of source file(s) related to the manifestation of the issue
3. **Location** of the affected source code (tag/branch/commit or direct URL)
4. **Step-by-step instructions** to reproduce the issue
5. **Proof-of-concept or exploit code** (if possible)
6. **Impact** of the issue, including how an attacker might exploit it

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt of your vulnerability report within 48 hours
- **Updates**: We'll keep you informed about our progress
- **Disclosure**: We'll work with you to understand and resolve the issue promptly
- **Credit**: We'll publicly thank you for your responsible disclosure (unless you prefer to remain anonymous)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Patch Release**: Depends on severity, typically within 30 days for high-severity issues

## Security Considerations for Users

### AWS Credentials

a9s requires AWS credentials to function. Please follow these best practices:

1. **Never commit AWS credentials** to version control
2. **Use IAM roles** when running on EC2 instances
3. **Use temporary credentials** when possible (STS, SSO)
4. **Follow principle of least privilege** - only grant necessary permissions
5. **Rotate credentials regularly**
6. **Enable MFA** on AWS accounts
7. **Use AWS profiles** to separate different environments

### Recommended IAM Policy

For read-only operations (recommended for most users):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "iam:Get*",
        "iam:List*",
        "s3:List*",
        "s3:GetBucketLocation",
        "s3:GetBucketTagging",
        "s3:GetBucketPublicAccessBlock"
      ],
      "Resource": "*"
    }
  ]
}
```

### Safe Usage

1. **Always use `--dry-run`** before destructive operations:
   ```bash
   a9s s3 cleanup --dry-run
   ```

2. **Review audit results** before taking action

3. **Test in non-production** environments first

4. **Keep a9s updated** to get latest security patches

5. **Use specific AWS profiles** for different environments:
   ```bash
   a9s --profile production ec2 list
   ```

### Data Privacy

- a9s does **not** collect or transmit any data
- All AWS API calls are made directly from your machine to AWS
- No telemetry or usage analytics
- Configuration and credentials remain local

## Security Features

### Current Security Features

- ✅ No credential storage in the application
- ✅ Uses official AWS SDK with secure defaults
- ✅ Supports AWS profiles and temporary credentials
- ✅ Read-only by default for most operations
- ✅ Dry-run mode for destructive operations
- ✅ No external dependencies on runtime

### Security Scanning

We regularly scan for:

- Known vulnerabilities in dependencies (`go mod audit`)
- Security issues in code (`gosec`)
- Code quality issues (`golangci-lint`)

You can run these checks yourself:

```bash
# Security scan
make sec

# Full quality check
make lint
```

## Known Security Limitations

1. **Terminal UI Security**: TUI displays sensitive AWS data in the terminal. Ensure your terminal is secure and not being recorded/shared.

2. **No Audit Logging**: a9s does not maintain its own audit log. Use AWS CloudTrail for comprehensive audit logging.

3. **Local Execution**: a9s runs with your local AWS credentials. Protect your local machine.

## Dependencies

We keep dependencies minimal and up-to-date:

- AWS SDK Go v2 (official AWS library)
- Cobra (CLI framework)
- BubbleTea (TUI framework)

All dependencies are scanned for vulnerabilities during CI/CD.

## Vulnerability Disclosure Policy

Once a vulnerability is fixed:

1. We'll release a security patch
2. We'll publish a security advisory on GitHub
3. We'll credit the reporter (unless anonymous)
4. We'll update this SECURITY.md with mitigation steps if needed

## Security Best Practices for Contributors

If you're contributing to a9s:

1. **Never commit secrets** (use `.gitignore`)
2. **Validate all user input**
3. **Use parameterized queries** for any data operations
4. **Follow secure coding guidelines**
5. **Add security-focused tests**
6. **Run security scanners** before submitting PRs:
   ```bash
   make sec
   make lint
   ```

## Questions?

If you have questions about security that don't involve reporting a vulnerability, please open a GitHub issue with the `security` label.

---

**Last Updated**: 2025-08-07
**Version**: 1.0
