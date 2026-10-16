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
