# Security Policy

## Reporting a Vulnerability

We take the security of YOLO seriously. If you believe you've found a security vulnerability, please responsible disclosure by following these guidelines.

### How to Report

1. **Do NOT create a public GitHub issue** for security vulnerabilities
2. Instead, report it privately via one of these methods:
   - Email maintainers directly (see below)
   - Use the private vulnerability reporting feature in GitHub

### What We Need

Please include as much information as possible:

- **Description**: Clear description of the vulnerability
- **Steps to Reproduce**: How to trigger the issue
- **Impact**: Potential consequences if exploited  
- **Proof of Concept**: Code or commands demonstrating the issue (if available)
- **Suggested Fix**: Your recommended mitigation (optional)
- **Environment**: Go version, OS, dependencies

### Response Timeline

- **Initial Response**: Within 48 hours
- **Acknowledgment**: We'll confirm receipt within 72 hours
- **Assessment**: We'll evaluate and categorize within 5 business days
- **Resolution**: Target timeframe depends on severity (see below)

### Severity Classification

| Level | Description | Response Time |
|-------|-------------|---------------|
| **Critical** | Remote code execution, authentication bypass, data exposure | 24-48 hours |
| **High** | Privilege escalation, information leakage | 3-5 business days |
| **Medium** | Denial of service, minor security issues | 10 business days |
| **Low** | Best practice violations, minimal impact | 30 business days |

### What to Expect

1. **Receipt Confirmation**: You'll receive acknowledgment of your report
2. **Assessment Update**: We'll inform you of our findings and planned action
3. **Regular Updates**: During investigation (every 5-7 days)
4. **Resolution Notification**: When the issue is fixed
5. **Credit**: Acknowledgment in release notes (with your permission)

### What We Won't Do

- Sue you or report to authorities for good-faith security research
- Harass or retaliate against security researchers
- Publicly disclose without your consent (unless legally required)

### Supported Versions

We currently provide security updates for:

| Version | Supported |
|---------|-----------|
| Latest Release | ✅ Yes |
| Previous Release | ⚠️ Critical only |
| Older Releases | ❌ No |

### Scope

**In Scope:**
- YOLO application vulnerabilities
- VE.Direct protocol parser (victron package)
- Configuration handling
- File operations and permissions

**Out of Scope:**
- Third-party dependencies (report to their maintainers)
- Denial of Service attacks
- Physical security issues
- Social engineering
- Issues in user-configured code

### Bug Bounty

At this time, we do not offer monetary rewards for vulnerability reports. However, we:

- Publicly acknowledge contributors who find significant issues
- Provide early access to security updates
- Offer to collaborate on the fix if desired

### Responsible Disclosure Agreement

By reporting a vulnerability, you agree to:

1. Not exploit or test the vulnerability beyond proof of concept
2. Give us reasonable time to fix before public disclosure (30 days minimum)
3. Communicate responsibly with other researchers about your findings

### Contact Information

**Primary Method**: Email maintainers  
- See LICENSE file for maintainer information

**GitHub Security Advisories**: Use the [Private Vulnerability Reporting](https://github.com/scottstg/yolo/security/advisories/new) feature

---

## Security Best Practices

When using YOLO, follow these security guidelines:

### Configuration Security
```bash
# Set appropriate file permissions on config.json
chmod 600 ~/.yolo/config.json

# Review sensitive data in configuration files
# Avoid hardcoding credentials or tokens
```

### Dependency Management
```bash
# Regularly update dependencies
go get -u

# Audit for known vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Bluetooth Device Security
```bash
# Only connect to trusted Victron devices
# Verify device MAC addresses in production environments
# Use secure channels for sensitive operations
```

### Environment Variables
```bash
# Use environment variables for sensitive configuration
export YOLO_API_KEY="your-secure-key"
```

## Incident Response

In the event of a confirmed security incident:

1. **Immediate Action**: We'll release an urgent patch if needed
2. **Notification**: Users will be notified via release notes and issue tracker
3. **Mitigation Guidance**: Documentation on protecting against the vulnerability
4. **Timeline Disclosure**: When the issue was discovered, fixed, and disclosed

## Security Audit History

| Date | Issue | Severity | Status |
|------|-------|----------|--------|
| TBD | First security audit pending | - | Not started |

---

**Thank you for helping keep YOLO secure!** 🔒
