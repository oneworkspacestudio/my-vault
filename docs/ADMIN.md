# Administration

One data root has exactly one supervisor lock. A Windows service and a portable
launch cannot run that same installation concurrently. `MyVault.exe manage`
opens the existing local owner tools rather than starting another engine.

```powershell
.\MyVault.exe run --data C:\MyVaultData
.\MyVault.exe manage --data C:\MyVaultData
.\MyVault.exe stop --data C:\MyVaultData
```

Service controls require appropriate Windows administrator rights:

```powershell
.\MyVault.exe service-install --data C:\MyVaultData
.\MyVault.exe service-start --data C:\MyVaultData
.\MyVault.exe service-stop --data C:\MyVaultData
.\MyVault.exe service-uninstall --data C:\MyVaultData
```

Service uninstall removes only registration. It never deletes vault state,
exports, attachments or backups. Deleting the data folder is a separate manual
destructive operation; make an independent verified backup first.

Service installation copies the launcher into
`%ProgramData%\OneWorkspaceMyVaultServices\<installation-id>\MyVault.exe`.
That local NTFS folder is owned by Administrators; SYSTEM/Administrators can
write and ordinary users can only read/execute. A service must not execute the
portable copy from a writable or exFAT program folder. Installing/updating this
copy requires elevation; stop the service first. Its protected binary remains
after uninstall, along with the vault data. Native elevated/service/reboot
acceptance remains a separate gate; portable startup requires no elevation.

Default persistent layout:

| Location | Purpose |
| --- | --- |
| `myvault.json` | Public URL/listener, retention and public backup recipient/settings |
| `myvault-sso.json` | Optional protected server-side OIDC client; excluded from portable recovery archives |
| `engine/db.sqlite3` plus WAL/SHM while running | Standard Vaultwarden SQLite state |
| `engine/attachments`, `engine/sends` | Vaultwarden encrypted content files |
| `engine/rsa_key*` | Server signing material; protected by NTFS ACLs |
| `runtime/<content-hash>` | Verified engine and web assets extracted from launcher |
| `recovery.json` | Private local backup/export status; no export password/content |
| `backups/*.age` or explicit independent destination | Encrypted server archives |
| `control.json` | Transient local capability; never export/publish or proxy |

Use One Workspace's verified updater for compatible app releases. It stops the
suite and snapshots state before replacement, with rollback on failed startup.
For standalone upgrades: stop, make/verify an encrypted full backup, preserve old
launcher, verify new release checksums, replace only the launcher, and restart
with the identical data root. Never downgrade an already migrated database;
restore its pre-upgrade snapshot alongside the old launcher if rollback is needed.

Keep all private state on local NTFS. Account access is independent of Windows
DPAPI; encrypted recovery archives use portable age secrets. Local control
capabilities are ephemeral, regenerated after restore/restart, and ACL-protected.
