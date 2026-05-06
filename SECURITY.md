# Security Policy

Shellin is terminal access software. Please report issues that could affect
terminal data, auth tokens, session attachment, release provenance, or update
integrity privately before opening a public issue.

## Report a Vulnerability

Send a private report to `contact@shellin.dev`. Include:

- affected version or commit
- steps to reproduce
- expected impact
- whether terminal data, auth tokens, session attachment, or update integrity is
  affected

## Scope

In scope for this repository:

- CLI PTY handling
- WebRTC terminal/control DataChannels
- signaling token validation and replay protection
- signaling peer replacement and detach behavior
- ICE server generation and filtering
- build provenance reporting
- signed release manifest and self-update validation

Out of scope for this repository:

- App Store purchase flows
- hosted infrastructure configuration
- private production credentials
- abuse and rate-limit policy details
- iOS app UI/UX
