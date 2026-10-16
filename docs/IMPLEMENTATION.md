# Implementation decisions

Pinned upstream Vaultwarden 1.37.4 has native Windows conditional code and a
bundled SQLite build feature. Upstream publishes no Windows asset for this tag.
Build the reviewed exact source with Rust 1.99.0/MSVC and bundled OpenSSL/SQLite;
do not substitute a third-party Windows EXE. Check the locked dependency audit,
native build and PE runtime dependencies before distributing.

The main MyVault.exe embeds a verified runtime ZIP. On first start it extracts
versioned files into the private NTFS data root, verifies their manifest on each
launch and holds the engine binary against replacement through its lifetime.
One lock covers the root; service and portable launches cannot own it together.
The Go supervisor and one Rust engine are separate processes. No Docker, external
database or separately installed web server is required. Persistent data are
outside the executable. Moving to another computer uses encrypted backup/restore.

Vaultwarden owns registration, client authentication, rate limits, MFA and APIs.
Its independent master-password route always remains enabled. MyAccounts is an
optional upstream OIDC sign-in provider using a dedicated vault client, never the
portal's client secret or session cookie. Workspace links to a distinct vault
origin; it does not proxy the vault client or inject shared navigation scripts.
Standalone works with Workspace/MyAccounts absent. See [Sign-in](SIGN-IN.md).

The Material-inspired theme changes CSS, logo markup and page titles. A small
first-party script adds the standard SSO link and starts that route on a fresh
Workspace launch that reaches login. It never reads credential fields, browser
vault storage, tokens, key material or Angular services. Every original upstream
JavaScript file remains byte-for-byte unchanged. First-party assets have content
hashes in filenames; HTML is served without caching to avoid stale upgrade UI.

Public requests reach a loopback-only wrapper through a configured HTTPS proxy.
Host validation and a fixed configured origin reject arbitrary proxy origins;
admin/diagnostics are blocked on the public listener. Initial registration is a
time-limited local owner action, restricted to one email address. No public open
signup default, disabled admin interface, no favicon fetching/telemetry/push by
default. Local owner management uses a separate loopback listener with a random
capability, exact Host/Origin checks and no cookies or persistent browser token.

Backups pause/drain public requests, stop the sole engine, use SQLite VACUUM INTO
plus integrity_check, and snapshot required attachments/keys/configuration.
Encryption uses standard age, with a passphrase or independent age recipient.
Verify by decrypting and checking every manifest digest plus SQLite integrity.
Restores require an empty private root and never overwrite another installation.
Recovery keys/passwords are never DPAPI-bound to the original computer.

The supplied brief's Windows/mobile/offline/total-loss requirements are recorded
in acceptance.json. Production publication is gated on actual evidence; native
same-host isolated deployment tests are not a clean-machine or iOS/Android test.
