# Security policy

## Reporting a vulnerability

Do not open a public issue for a security problem.

Report it privately through GitHub's [private vulnerability
reporting](https://github.com/vivaui/viva/security/advisories/new) on this
repository, which keeps the report and the discussion around it private until an
advisory is published. If that link is unavailable, contact a maintainer
privately — through their GitHub profile rather than a public thread — and say
which project the report concerns.

Please include what you need to reproduce it, which platform you saw it on, and
what an attacker gains. A short description and the offending code path is
enough to start; you do not need a working exploit.

We aim to acknowledge a report within a few days and to say whether we consider
it a vulnerability. Because this is a pre-1.0 project maintained in the open,
fixes land as ordinary commits on `main` rather than as backports to older
versions.

## What counts

Viva runs with the privileges of the application that uses it, in the same
process. Reports we treat as security issues include:

- Memory-safety faults reachable from input: bounds errors, use-after-free, or
  integer overflows in the text, layout, scene, or image paths.
- Parsing bugs reachable from untrusted data: TOML configuration, PNG and icon
  loading, clipboard and drag-and-drop payloads, and platform text input.
- Escapes from the intended boundary between the application and the framework:
  for example a component or clipboard path that reads memory it does not own, or
  a headless control channel that can be made to run commands it should not.
- Anything in a released binary that weakens the process sandbox or hardening the
  platform would otherwise provide.

## What does not

- Vulnerabilities in a third-party library, unless Viva calls it in a way that
  makes them reachable; please report those upstream too.
- Bugs that require the attacker to already control the machine, the user's
  account, or the application's own binary.
- Missing features, performance problems, and the absence of a hardening flag
  that is not claimed in the documentation.

## Supported versions

Viva has no tagged release yet. Before the first release, only `main` is
supported, and fixes land there.
