# Security Policy

## Our Commitment

Security is a top priority for the Datadog CLI project. We take all security vulnerabilities seriously and appreciate the security community's efforts to responsibly disclose issues.

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

Once version 1.0.0 is released, we will support:
- Current major version (e.g., 1.x.x)
- Previous major version for 6 months after new major release

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

### How to Report

**Option 1: GitHub Security Advisories (Preferred)**

1. Go to the repository's Security tab
2. Click "Report a vulnerability"
3. Fill out the vulnerability report form
4. Submit the report

**Option 2: Email**

Send an email to: **[INSERT SECURITY EMAIL]**

Include the following information:

```
Subject: [SECURITY] Brief description

### Vulnerability Details
- Type of vulnerability (e.g., RCE, XSS, privilege escalation)
- Affected component/file
- Version(s) affected
- Impact assessment (who is affected, severity)

### Steps to Reproduce
1. Step one
2. Step two
3. ...

### Proof of Concept
(Code, screenshots, or other evidence)

### Suggested Fix
(If you have one)

### Your Contact Information
- Name (optional)
- Email
- GitHub username (optional)
```

### What to Expect

**Within 24 hours**: We will acknowledge receipt of your report.

**Within 7 days**: We will provide an initial assessment of the report, including:
- Confirmation whether it's a valid security issue
- Severity assessment
- Expected timeline for a fix

**Regular updates**: We will keep you informed of our progress at least every 7 days.

**Resolution**: Once a fix is ready, we will:
1. Notify you before public disclosure
2. Credit you in the security advisory (if desired)
3. Coordinate disclosure timing with you

## Response Timeline

| Stage | Timeline |
|-------|----------|
| Acknowledgment | Within 24 hours |
| Initial assessment | Within 7 days |
| Regular updates | Every 7 days |
| Fix development | Varies by severity |
| Public disclosure | Coordinated with reporter |

## Severity Levels

We use the CVSS v3.1 framework to assess severity:

### Critical (9.0-10.0)
- Remote code execution
- Privilege escalation to admin
- Data exfiltration of all data
- Complete system compromise

**Fix timeline**: Immediate (within 7 days)

### High (7.0-8.9)
- Authentication bypass
- SQL injection
- Significant data exposure
- Denial of service affecting all users

**Fix timeline**: Within 14 days

### Medium (4.0-6.9)
- Limited privilege escalation
- Cross-site scripting (XSS)
- Information disclosure (limited)
- Denial of service affecting some users

**Fix timeline**: Within 30 days

### Low (0.1-3.9)
- Minor information disclosure
- Security configuration issues
- Low-impact vulnerabilities

**Fix timeline**: Within 90 days

## Disclosure Policy

### Coordinated Disclosure

We follow a **coordinated disclosure** policy:

1. **Report received**: Vulnerability reported privately
2. **Investigation**: We confirm and assess the issue
3. **Fix developed**: We create and test a patch
4. **Pre-disclosure**: We notify the reporter before release
5. **Release**: We publish the fix and security advisory
6. **Public disclosure**: Typically 7 days after fix release

### Public Disclosure Timeline

- **For critical vulnerabilities**: 7 days after fix release
- **For other vulnerabilities**: 14 days after fix release
- **Extended timeline**: If circumstances require (e.g., complex fixes, dependency issues)

### What We Disclose

Security advisories will include:
- Vulnerability description
- Affected versions
- Fixed versions
- Severity rating
- Credit to reporter (if permitted)
- Mitigation steps (if fix not yet available)

## Security Best Practices

### For Users

**1. Keep Updated**
```bash
# Check current version
dd --version

# Update via Homebrew
brew upgrade datadog-cli

# Update via package manager
apt-get update && apt-get upgrade datadog-cli  # Debian/Ubuntu
yum update datadog-cli                          # RedHat/CentOS
```

**2. Protect Your API Keys**
```bash
# Use environment variables
export DD_API_KEY="your_key_here"
export DD_APP_KEY="your_app_key_here"

# Never commit keys to version control
echo "DD_API_KEY=*" >> .gitignore
echo "DD_APP_KEY=*" >> .gitignore

# Use restrictive file permissions for config files
chmod 600 ~/.dd-config.yaml
```

**3. Verify Downloads**
```bash
# Check SHA256 checksums
sha256sum dd-linux-amd64
# Compare with published checksums

# Verify GPG signatures (when available)
gpg --verify dd-linux-amd64.sig dd-linux-amd64
```

**4. Use Least Privilege**
```bash
# Use read-only API keys when possible
# Create dedicated API keys for the CLI
# Rotate keys regularly
```

### For Contributors

**1. Secure Coding Practices**

- Validate all user input
- Sanitize data before output
- Use prepared statements for any queries
- Avoid command injection vulnerabilities
- Handle errors securely (don't expose sensitive info)

**2. Dependency Management**

```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Update dependencies regularly
go get -u ./...
go mod tidy
```

**3. Code Review**

- All changes require review
- Security-sensitive changes require two reviews
- Automated security scans run on all PRs

**4. Testing**

```bash
# Run security tests
gosec ./...

# Check for common vulnerabilities
go vet ./...

# Run with race detector
go test -race ./...
```

## Known Security Considerations

### API Key Handling

The CLI handles sensitive Datadog API keys. We implement:

- **No logging of keys**: API keys never written to logs
- **Environment variables**: Preferred method for key storage
- **Memory clearing**: Keys cleared from memory after use
- **No persistent storage**: Keys not stored in files by default

### Network Security

- **HTTPS only**: All API calls use HTTPS
- **Certificate validation**: TLS certificates validated
- **No insecure protocols**: HTTP, FTP, Telnet not used
- **Timeout handling**: All network calls have timeouts

### Input Validation

- **Command injection prevention**: All shell commands escaped
- **Path traversal prevention**: File paths validated
- **Query injection prevention**: API queries sanitized
- **Integer overflow checks**: Numeric inputs validated

## Security Features

### Built-in Protections

**1. Secure Defaults**
- HTTPS enforced for all API calls
- TLS 1.2+ required
- API key validation before use
- Automatic timeout on long operations

**2. No Elevated Privileges**
- CLI runs with user privileges
- No setuid/setgid binaries
- No root/admin requirements
- Minimal file system access

**3. Minimal Attack Surface**
- Single static binary
- No external dependencies at runtime
- No plugin system (yet)
- No eval/exec of user code

### Future Security Enhancements

Planned security improvements:

- [ ] GPG signature verification for releases
- [ ] SBOM (Software Bill of Materials) generation
- [ ] Runtime security scanning integration
- [ ] Hardware security key support
- [ ] Config file encryption
- [ ] Audit logging for sensitive operations

## Vulnerability History

No security vulnerabilities have been reported or fixed yet.

When vulnerabilities are discovered and fixed, they will be listed here with:
- CVE identifier (if assigned)
- Affected versions
- Fixed version
- Severity rating
- Brief description

## Security Tools

### Automated Scanning

We use the following tools to identify vulnerabilities:

**Static Analysis**:
- golangci-lint (multiple security linters)
- gosec (Go security checker)
- go vet (Go built-in analyzer)

**Dependency Scanning**:
- GitHub Dependabot
- nancy (OSS Index integration)
- go list (module analysis)

**Container Scanning** (if using Docker):
- Trivy
- Snyk
- Clair

### CI/CD Security

All pull requests automatically:
- Run security linters
- Check for known vulnerabilities
- Scan dependencies
- Verify code signing (planned)

## Contact

### Security Team

For security issues: **[INSERT SECURITY EMAIL]**

For security questions: Open a GitHub Discussion in the Security category

### PGP Key

```
[INSERT PGP PUBLIC KEY BLOCK WHEN AVAILABLE]
```

## Recognition

We believe in recognizing security researchers who help make our project safer:

### Hall of Fame

Security researchers who have responsibly disclosed vulnerabilities:

- (None yet - be the first!)

### Acknowledgments

We thank the security community for:
- Responsible disclosure of vulnerabilities
- Suggestions for security improvements
- Contributions to security documentation

---

## Resources

**Security Guides**:
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security](https://go.dev/security/)
- [CWE Top 25](https://cwe.mitre.org/top25/)

**Reporting Guidelines**:
- [NIST CVD Guide](https://www.nist.gov/itl/executive-order-improving-nations-cybersecurity/vulnerability-disclosure-guidelines)
- [ISO 29147](https://www.iso.org/standard/72311.html)

---

**Thank you for helping keep Datadog CLI and its users safe!**

---

**Version**: 1.0
**Last Updated**: January 22, 2026
