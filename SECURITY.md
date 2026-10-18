# Security Policy

goki reads and writes files on your computer, so we take security seriously.

## Supported versions

Security fixes are released for the latest minor version of goki.

## Reporting a vulnerability

**Please don't report security issues in public issues.**

Use GitHub's [private vulnerability reporting](https://github.com/jdtobe/goki/security/advisories/new) instead. Include:

- the goki version (`goki version`) and your operating system,
- a description of the issue and its impact,
- steps to reproduce, ideally with a minimal wiki folder.

You can expect an acknowledgement within a few days. Once a fix is ready we'll coordinate disclosure with you and credit you in the release notes unless you'd prefer otherwise.

## Scope

goki's threat model and built-in protections are described in [Sharing and Security](docs/sharing-and-security.md). Issues of particular interest:

- reading or writing files outside the served folder,
- cross-site or DNS-rebinding attacks against a goki running on localhost,
- script execution through rendered Markdown or served files,
- authentication bypass when `auth` is configured.

Running goki on an untrusted network without TLS or authentication is outside its supported configuration.
