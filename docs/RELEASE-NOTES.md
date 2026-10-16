# 0.2.0-rc.2 — local production acceptance candidate

First-run testing found an OS dark preference could leave dark dialogs/cards
under the light branding. The complete compatible-client light palette now
covers these surfaces and native form controls, without changing client settings
or cryptographic JavaScript. Corrected the recovery page's restore command to HTTPS.

A disposable account was registered with the official SDK, then used in the
browser to save a login. The official CLI independently decrypted it. Replacing
the executable and reusing the private data folder preserved the account, original
password, item and local certificate. Browser unlock and further editing passed.
This is same-host executable replacement evidence, not a clean-host reinstall.
The final browser registration submission remains outside automated UI acceptance.

Material-inspired My Vault branding now covers the signed-in sidebar, login,
unlock, loading and SSO connector screens, dynamic page titles and owner tools.
Content-versioned assets and uncached HTML prevent stale styling after upgrades.
All original upstream JavaScript remains byte-for-byte unchanged.

The updated Workspace manager configures optional native upstream MyAccounts
OIDC with a dedicated client and exact separate-origin callback. Existing
verified-email accounts and repeat MyAccounts session reuse were exercised in
the browser. Vault unlock remains separate. Password login works with MyAccounts
stopped, and recovery excludes the optional identity configuration.

Corrected restore-wizard HTTPS defaults and invalid/expired certificate guidance.
Windows service installation now requires elevation and copies the supervisor
to an administrator-owned NTFS machine folder resolved through the Windows known
folder API. It refuses a pre-created ordinary-user-owned service folder and
does not execute the portable binary as SYSTEM from an exFAT/writable location.
Elevated lifecycle/reboot validation remains pending.

The final candidate repeated official CLI sync/decrypt, offline unlock,
independent encrypted export/import, verified full backup/restore and exact
attachment recovery with MyAccounts/original source absent, on this same host.

Production eligibility remains false until the actual remaining gates in
[PRODUCTION-GATES.md](PRODUCTION-GATES.md) are closed. No production release is
claimed by this candidate and no existing published binary is replaced.

## Earlier preview foundation

My Vault packages a native Windows Vaultwarden server, compatible branded web
vault, local HTTPS, independent accounts and recovery tools. No Docker, separately
installed database, custom vault encryption or GitHub Actions.

Verified with synthetic data: official Bitwarden CLI 2026.9.1 login/sync/decryption,
logins/secure notes/folders/search/generator/TOTP, attachments, password-protected
encrypted export imported into a different account, offline cache unlock with
server stopped, full age-encrypted recovery into a fresh data root and exact
attachment bytes with the source server/Workspace/MyAccounts absent.

The native engine is Vaultwarden 1.37.4 with a reviewed dependency-only rustls
0.23.46 patch. Cryptographic client JavaScript is unchanged. Source, exact locks,
licenses, build receipt, audit findings and checksums accompany the Windows package.

**Unsigned preview, not production ready.** Clean-computer recovery, Windows
desktop/extension, browser registration/sign-in after explicit Windows certificate
trust, service/reboot, MFA/passkeys/collections and independent security review
remain unperformed. Mobile work is deferred. Read acceptance.json and
SECURITY-REVIEW.md; do not store important credentials in this preview.

Run MyVault.exe on Windows x64 with a protected local NTFS data folder. Explicitly
trust its unique localhost-only certificate through local owner tools before
browser sign-in. Official CLI local use needs NODE_EXTRA_CA_CERTS pointing at that
certificate. Remote clients require a separate publicly trusted HTTPS vault origin.
Keep independent master/MFA recovery, backup and export secrets off the server.
Attachments are recovered by full server backups, not assumed included in JSON.

The independent app update contract targets the new One Workspace vault entry;
fresh Workspace installations include it. Existing installations retain their
selected app set and do not silently gain a new vault or change authentication.
