# Security policy

Please report vulnerabilities privately through GitHub's **Report a
vulnerability** security-advisory form for this repository. Do not open a
public issue containing exploit details, credentials, private endpoints, or
data from a guest machine.

This project is an experimental MVP. Deployments should keep the broker on
loopback behind authenticated TLS ingress, generate independent release and
administrator secrets, use short session lifetimes, and review
`docs/THREAT_MODEL.md` before public exposure.
