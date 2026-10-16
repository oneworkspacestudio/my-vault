# Production acceptance

The current build is a release candidate. `productionEligible` stays false until
the remaining evidence is recorded. Do not use an RC label as a production claim.
Mobile applications and mobile acceptance are deferred at the owner's request.

## Implemented and exercised

- Native Windows build, protected persistent NTFS state and verified runtime.
- Compatible browser login, item creation, CLI/browser sync and lock/unlock.
- Disposable first-account registration through the official SDK; executable
  replacement retaining the same private root, account, item and certificate.
- Native MyAccounts OIDC with standard PKCE, verified-email linking of existing
  vault accounts, repeat identity-session reuse, and separate vault unlock.
- Independent official-client login without Workspace/MyAccounts; same-host
  offline unlock and encrypted restore/export/import checks.
- Encrypted database/attachment recovery, corruption rejection, multi-user
  isolation, management-origin protection, rate limiting and TLS checks.
- Go test/vet and dependency-symbol scan, with upstream findings disclosed.

Exact scope and final candidate regression results are in `acceptance.json`.

## Evidence still required

| Gate | Required evidence | Current limitation |
| --- | --- | --- |
| Browser first-account enrollment | A person completes the stock browser registration | Native SDK registration and subsequent browser sign-in/save/read passed; browser Create account submission remains unperformed under the new-credential UI handoff rule |
| Official desktop / browser extension | Sign in, sync, lock, disconnect, unlock/read a cached item | Desktop password-manager UI is excluded by the Computer Use skill; manual check required |
| Clean Windows recovery | On a second clean machine or VM, restore from only independent archive/secrets and inspect items/attachments; import portable export | No clean VM/second Windows host is available in this session; a new data root on this host is insufficient |
| Windows service lifecycle | Elevated install/start/stop/uninstall, reboot, direct access and retained-data check | Current process is not elevated; rebooting the owner's active machine is not an automated substitute |
| Security sign-off | Review optional OIDC, first-party presentation code, origin/ACL/backup boundaries and disclosed upstream advisories | No independent security reviewer has signed off |

Passkeys, organization/shared-collection migration, SSH formats and MFA login are
not part of the verified feature claims yet. Verify each before advertising it.
Attachment preservation is verified in the full server backup, not portable JSON.

## Safe validation process

Use synthetic data and fresh protected NTFS folders. Never reinstall over the
owner's live Workspace or delete its data to run acceptance. For service checks,
use the candidate binary and a separate data root, run commands from an elevated
PowerShell, and retain the root after `service-uninstall`. The available commands
are `service-install`, `service-start`, `service-stop`, `service-uninstall` with
`--data YOUR_TEST_ROOT`. Complete a reboot check on a disposable test machine.

After a clean-host restore, use the direct vault URL and independent master
password while the original server and MyAccounts are absent. Independently
verify the known item values and attachment bytes. Then lock the official client,
disconnect the network, and unlock/read its retained cache without logging out.

Do not enable SSO-only login, automatically trust certificates, disable TLS
validation, store a master password in Workspace, or change encryption to make
any acceptance step pass. Publish a production build only after actual gate
evidence is reviewed; never edit existing published binary assets in place.
