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
Its independent master-password route always remains enabled. MyAccounts is only
the ecosystem launcher identity; the vault does not receive Workspace client
secrets. Workspace links to a distinct vault origin; it does not proxy the vault
client or inject shared scripts. Standalone works with Workspace/MyAccounts absent.

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
