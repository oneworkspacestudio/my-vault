# Recover without the original server

## Prepare your independent recovery kit

Keep these outside One Workspace, MyAccounts and this vault:

- Master password and MFA recovery code, secured appropriately.
- An encrypted full server backup and its passphrase or age secret identity.
- A password-protected encrypted JSON vault export and its separate password.
- A verified copy of MyVault.exe/checksums, or instructions for an independent
  compatible Vaultwarden/Bitwarden deployment.
- Copies of required attachments and organisation exports, where ordinary
  individual-vault export does not include them.
- This guide and the exact client/server versions used for your recovery check.

No server backup can decrypt an unknown master password. MyAccounts passwords
cannot replace vault passwords. An age public recipient cannot decrypt backups;
its matching secret identity must survive outside the server.

## Full-server recovery on another Windows computer

1. Obtain a verified launcher and the independent `.age` backup. You need no
   access to the original server, Workspace or MyAccounts.
2. Choose a new empty NTFS folder. Existing installations are never overwritten.
3. Run:

   ```powershell
   .\MyVault.exe restore --data C:\MyVault-Recovered --input X:\Recovery\myvault.age
   ```

4. The local-only wizard asks for the backup passphrase or age secret identity.
   It decrypts privately, validates the manifest and every file, and checks SQLite.
5. Start `MyVault.exe run --data C:\MyVault-Recovered`. Explicitly trust the new
   localhost certificate for your Windows account in owner tools, then open
   the secure local vault. Sign in with your vault
   email/master password and MFA. Inspect actual item contents and attachments.
6. Reconfigure a dedicated valid HTTPS hostname for remote clients. Update each
   client's self-hosted server URL and sync. Test offline unlocking again.
7. Scheduling is disabled after restore until you select independent storage on
   the new computer. Enrollment is closed. Preserve the source backup.

The data under `engine` are standard Vaultwarden state. For independent upstream
deployment, decrypt with the official `age` tool and unpack the ZIP on a protected
filesystem, verify `manifest.json`, and use its `engine` directory as Vaultwarden's
DATA_FOLDER. Use the pinned engine or an explicitly compatible newer version.
Create new host configuration; do not carry over the local management capability.

## Portable-export recovery into a different account

1. In the compatible client choose Tools → Export vault → encrypted JSON →
   **Password protected**. Keep its independent export password.
2. Create a fresh independent compatible account, open Tools → Import data and
   choose Bitwarden JSON. The official client decrypts the export with its password.
3. Inspect representative login passwords, secure notes, custom fields and TOTP
   entries. Compare item counts and organisation ownership.
4. Recover excluded attachments/organisation items separately. Import is not a
   deduplicating sync operation; importing twice may duplicate items.

Recording an export in My Vault verifies only `encrypted`/`passwordProtected`
format markers and records a fingerprint/date. It does not prove the password,
successful import, or completeness. No plaintext export is accepted or uploaded.

## Offline survival

Install a compatible desktop/extension client, sign in, sync, lock it, disconnect
the network, and unlock/read an item. Repeat on an independent device before
relying on this. Stay logged in. Logout, reinstall, device changes, vault timeout
configured as logout, lost local data or platform restrictions may remove access.
Do not assume offline editing or sync. The web vault is not an offline client.

## Backup consistency and retention

My Vault blocks/drains requests and stops its sole engine before taking a cold
snapshot. SQLite `VACUUM INTO` reads committed WAL state into a standalone database.
The snapshot includes required files; logs, temporary files and runtime binaries
are excluded. Standard age encrypts it before it reaches offsite/removable storage.
Passphrase backups are decrypted and integrity-checked immediately. Public-key
scheduled backups need the independent private identity for that final check.

Retention is an explicit owner action, preserves unverified archives, verifies
recorded archive hashes, and keeps at least two verified backups. A file whose
presence is confirmed is not automatically labelled recoverable.
