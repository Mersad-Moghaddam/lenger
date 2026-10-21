# Security policy

Lenger is pre-release and has no supported stable versions yet. The Docker
socket provides root-equivalent control. Treat the app as privileged software.

Do not submit real credentials, registry tokens, Docker config files, environment
values or sensitive image history in issues, pull requests, screenshots or logs.

For vulnerabilities, use GitHub's private vulnerability reporting for this
repository if enabled. If it is unavailable, open a minimal issue requesting a
private reporting channel without disclosing exploit details or secrets.

Default safeguards include non-forced image removal, reference rechecks, masked
inspection and no credential persistence. They still require runtime verification
before production use. Credential helpers are executables from PATH: install
only helpers you trust.
