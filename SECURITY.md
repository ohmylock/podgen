# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in podgen, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email the maintainers directly or use GitHub's private vulnerability reporting feature
3. Include details about the vulnerability and steps to reproduce

We will respond within 48 hours and work with you to understand and resolve the issue.

## Security Best Practices

When using podgen:

- Store S3 credentials securely (use environment variables, not config files in version control)
- Keep `podgen.yml` out of public repositories if it contains secrets
- Use S3 bucket policies to restrict access to your podcast files
- Review uploaded content before making feeds public
