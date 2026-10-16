# Security policy

## Supported versions

Fixes go to the latest tagged release and to `code`. Older tags get a fix only
when the issue is severe and the change is small.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Use GitHub's private report: open
<https://github.com/ows4444/tui/security/advisories/new> (the repository's
Security tab, "Report a vulnerability"). Say what you found, which version or
commit, and how to reproduce it. A terminal that is sent crafted bytes (an
escape sequence in text a program prints, for example) is in scope.

You can expect a first reply within 7 days. A confirmed issue is fixed on a
private branch, released, and then described in the advisory and in
[CHANGELOG.md](CHANGELOG.md), with credit to the reporter if they want it.

## What is in scope

- Output that lets untrusted text change the terminal's state (cursor, title,
  clipboard, hyperlinks) when a widget is documented to sanitise it.
- A crash, hang or unbounded allocation reachable from input bytes or from a
  widget's documented API.
- A way to run code or read files through the library itself.

The library imports only the standard library, so there are no third-party
dependencies to report here.
