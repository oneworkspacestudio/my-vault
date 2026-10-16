# Threat model

Protect: encrypted vault data, account metadata, attachments, server signing keys,
backup secrets, master passwords and client-side encryption keys.

Trust boundaries:

- Compatible clients own key derivation/encryption/decryption. My Vault does not
  implement a vault protocol or receive the user's master password.
- Vaultwarden owns API authorisation, authentication, MFA and encrypted records.
  A compromised vault origin or client can steal unlocked secrets; trusted HTTPS,
  a dedicated origin and current reviewed upstream versions remain essential.
- Local OS owner/admin/SYSTEM can access server state and local management. The
  gateway/admin capability is not a defence against a compromised OS admin.
- Public reverse-proxy clients are untrusted. Management has a different loopback
  listener, exact Host/Origin/capability checks, CSP, no cross-origin allowance,
  no persistent token/cookie, and no public admin/diagnostic exposure.
- Enrollment is an explicit short local-owner window for one email address.
  Public creation stays blocked; a master password is entered only in the client.
- Offsite/removable storage sees age-encrypted server archives. The private
  decryption key/passphrase must be held independently of the original machine.
  Checksums provide corruption detection, not an independent publisher identity.
- MyAccounts/Workspace are launcher/supervision conveniences. They never derive
  vault keys and are not required by standalone access, independent export import
  or restoration to a clean deployment.

Defaults: loopback listeners; remote HTTPS required; admin disabled; closed
registration; no inherited SMTP/admin/OIDC environment; favicon downloads,
telemetry and cloud push disabled; upstream login limits; no plaintext request
logging. Application logs omit raw upstream output and all request bodies.

Residual risks: unsigned Windows binaries, upstream/client vulnerabilities,
malicious compatible clients/browser extensions, stolen master/backup passwords,
compromised administrator, weak user passwords/MFA choices, untested client formats,
shared proxy rate-limit buckets, untested hardware/service/reboot acceptance.
Recovery is only as good as an independently retained secret plus verified archive
and tested account unlock. Never label an unperformed check passed.
