# Security review — 16 October 2026

This is an unsigned preview, not an independently audited password manager.
Use synthetic data until the remaining acceptance gates are completed.

The application source is Vaultwarden 1.37.4, commit recorded in upstream.lock.json.
The only engine change is the reviewed Cargo.lock update from rustls 0.23.45 to
0.23.46. That patch addresses the October rustls advisories in the newer TLS stack.
The exact replacement lock is supplied in upstream/Cargo.lock and the source
archive. No vault algorithm, protocol or compatible-client JavaScript is changed.

`cargo audit` initially reported ten dependency/advisory entries. After the
rustls patch, six entries remain in the lock and active dependency graph:

| Dependency | Advisory | Assessment and boundary |
| --- | --- | --- |
| rsa 0.9.10 | RUSTSEC-2023-0071 | Upstream Marvin timing advisory has no patched stable version. Vaultwarden uses RSA for JWT signatures/verification; My Vault exposes no RSA decryption API. Optional OIDC uses upstream authorization-code/PKCE and signed-token verification. This is a source/configuration assessment, not a timing audit. |
| rustls 0.21.12 | RUSTSEC-2026-0346, RUSTSEC-2026-0349 | Introduced by Rocket's optional direct TLS stack. The child receives a restricted environment, no ROCKET_TLS setting or Rocket configuration file, and serves HTTP only on loopback. Local TLS terminates in Go; public TLS in the separately configured trusted proxy. No custom rustls client session logging is added. |
| rustls-webpki 0.101.7 | RUSTSEC-2026-0098, RUSTSEC-2026-0099, RUSTSEC-2026-0104 | Same unused Rocket TLS stack. Do not enable native Rocket TLS or add unreviewed child environment settings. |

The findings are retained rather than suppressed. Track upstream dependency
updates and repeat this review before production deployment or enabling optional
features. The source assessment does not establish that every possible upstream
feature is unaffected. Maintainer/security review and outstanding platform/client
acceptance remain production gates.

Go 1.27.2 tests, vet and govulncheck found no vulnerable imported package or
reachable symbol. GO-2026-5932 is present at module level for the unused,
unmaintained x/crypto/openpgp package; this application does not import it.

Optional MyAccounts integration never enables SSO-only authentication,
unverified-email linking or public SSO registration. It links existing accounts
with upstream verified-email checks, then retains the upstream issuer/subject
association. Client secrets stay in protected NTFS configuration, never public
bootstrap responses or presentation code. Restored backups start without the
integration file. Invalid optional configuration falls back to independent
sign-in with a local owner warning. SSO does not decrypt the vault. Workspace
logout cannot erase another origin's vault session. Normal upstream lock controls
remain in force; no auto-unlock key store or custom key connector is introduced.
Provider-authentication-only sessions use the upstream vault lifetime after
sign-in; revoke vault sessions separately when removing access.

Native checks use synthetic accounts and isolated NTFS roots. They cover official
CLI login/sync/decryption, independent encrypted export/import, offline unlock,
full encrypted server restore and attachment bytes, cross-user item denial,
management capabilities/origins, duplicate ownership and certificate validation.
Unit checks additionally reject wrong recovery secrets, modified archives and
plaintext/account-restricted export records. Exact open checks are in acceptance.json.

Local certificate trust is explicit and per Windows user. The generated leaf is
restricted to localhost/127.0.0.1, CA=false, server-auth only, with no certificate
signing usage. No startup action changes Windows trust. Remote devices require
publicly trusted HTTPS, not this localhost leaf. Do not bypass browser warnings.

Primary advisory references: https://rustsec.org/advisories/RUSTSEC-2023-0071.html,
https://rustsec.org/advisories/RUSTSEC-2026-0346.html and the numbered RustSec
entries above. Audit-tool metadata and exact dependency sources accompany the
release; no user vault, master password or recovery secret is published.
