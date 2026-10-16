# My Vault implementation — in progress

User supplied the implementation brief, requested building the app, creating a
public oneworkspacestudio repository, integrating the Workspace installer and
updating its README. Mobile work was explicitly deferred afterward. Independent
master-password access, portable export and full encrypted recovery remain core.
No new vault protocol/cryptography, no Actions, no live vault/account test data.

Created https://github.com/oneworkspacestudio/my-vault (public), Actions disabled,
local source at I:/Projects/my-vault/main; source not yet committed/pushed. Existing
Workspace is at I:/Projects/one-workspace, current source base090c414/preview24.
Preserve its unrelated untracked installed-preview22 history. Live suite is not
modified by this build. Large artifacts are under tooling/my-vault; NTFS compiler
tools/builds/fixtures under the single named C:/Projects/.cache/my-vault-build.

Pinned unmodified Vaultwarden1.37.4 commit7c188967e7a467b19b2757bc3326f9cf3ccae327,
Rust1.99.0, native MSVC bundled SQLite/OpenSSL. Upstream has no Windows EXE asset.
Reviewed current upstream release/advisories, Windows cfg and bundled features.
Compatible web release v2026.7.0 SHA002e972bf0d0487ec0324b06d916de33e29de4c29bffd92ee3b843084c300570.
Scripts source fac81a4f35ede45965b220daa3255994abbf0eb1; client source branch commit
d2a095eb800fe80a9bc524579b974796eae51994 in vaultwarden/vw_web_builds. Build script
uses the OSS selfhost target; root license defaults GPL except bitwarden_license.
Retain corresponding source/original licenses; do not distribute restricted
commercial builds or change crypto JavaScript. Additive title/CSS/SVG/recovery link.

Go implementation exists: single embedded-runtime launcher, manifest verification,
deny-replacement engine handle, one root lock, loopback gateway, separate local
capability/Host/Origin-protected owner UI, no server admin/telemetry/SSO dependency.
Default closed registration; local ten-minute single-email enrollment through the
compatible client. Local API environment rewriting permits independent loopback
access when the public origin is unavailable. HTTPS required for remote config.
Vault's browser origin must differ from Workspace to isolate stored client keys.
Native service controls preserve data on uninstall; service startup waits for ready.

Backups drain requests/stop engine, VACUUM INTO committed WAL, copy required
attachments/signing keys/settings, age-encrypt and manifest/SHA/SQLite verify.
Passphrase backups verify by decrypting; public-recipient scheduled archives stay
unverified pending independent key verification. Restore wizard requires an empty
new NTFS root and an independent secret, resets enrollment/scheduling, defaults to
new local8914 URL. Export recorder accepts only password-protected encrypted JSON,
retains fingerprint/date only and never claims an import was verified. Explicit
retention keeps at least two verified copies; no auto-removal of unverified backups.
Public recovery guide clearly separates offline cache, export and server backup.

Source is not yet compile/native acceptance verified. Embedded payload files are
ignored and absent until a real native engine completes. Scripts build-engine.ps1,
package-runtime.py and build.ps1 prepare exact-source receipt, unchanged JS checks,
manifest/ZIP and launcher. SVG/ICO added, Windows syso generated (ignored). Docs
and acceptance.json retain pending statuses; no production claims or release yet.

Build state: initial Git Perl failed (missing Locale::Maketext::Simple). Installed
official rustup/Rust1.99 and verified StrawberryPerl5.42.3.1 portable (SHA in release
metadata), into the named NTFS cache. A dynamic-CRT build targets I:/tooling/my-vault/
rust-target; parallel preferred static-CRT explicit x86_64 Windows target builds
under the named C:cache/static-target. Both are compiling bundled OpenSSL; logs
engine-build.log and static-engine-build.log. Tooling build-engine.cmd and
build-static-engine.cmd capture fixed environments; no engine source edits.
Prefer completed static build and inspect PE dependencies before release.

Official Windows Bitwarden CLI2026.9.1 downloaded/digest verified outside source
under tooling/my-vault/bw-cli. No synthetic vault/account has been created yet.
No mobile/clean physical host/service/reboot acceptance. WindowsSandbox/Get-VM/
VBoxManage are not available by command lookup; never claim clean-machine tested.

Workspace edits prepared: vault group/package ownership, port base+14, no OIDC or
dependencies, launcher-only /vault route to separate HTTPS origin or local setup
instructions, exclusion from portal session gate, logo/overview/README/docs, build
arguments and org asset extraction. build-org invocation includes MyVault; its
lock pin/hash still must be added after an eligible component release. Contract
check requires13apps. No package built or release published yet. Workspace tests
passed once before its newly added vault-specific test file; rerun final tests.

Next: finish native engine, receipt and runtime payload; compile/fix Go, run unit
checks and real compatible client/browser create/sync/unlock, offline CLI, portable
encrypted export/import, total-source-server-loss restore/attachments, auth/TLS/
corruption/rollback checks in disposable NTFS roots. Review UI with actual browser.
Then complete notices/source/acceptance, publish eligible component (honest preview
limits), pin exact org asset, build/verify complete Workspace installer and README,
push/release only after required recovery gates, leave live installation unchanged.

## Continuation: native implementation and recovery exercised

The initial static engine completed (42,174,976 bytes; SHA7f349b6f2b977388717a151825799f78848e9af5329e612316d7f7a900d7a426).
Only system DLLs are required. Fixed explicit SQLite URL, disabled icon-fetch
configuration, dynamic engine proxy target after maintenance, deferred cleanup of
failed decrypt stages, shutdown/scheduler/management synchronization and pending
settings. Go tests/vet pass. Official CLI2026.9.1 actually logged in, synced,
created/decrypted login and secure note, folder/search/generator/TOTP, attachment
upload/download. Official SDK was used only to seed synthetic legacy-compatible
accounts; its newest V2 registration format is not supported by this server.
No product crypto/protocol changes or private/live user data are involved.

Password-protected official CLI JSON export imported into another account/server
with a clean profile; passwords/notes/TOTP compared, attachments excluded. age
server backup decrypted/hash/SQLite verified. Original server stopped, CLI locked
then unlocked/decrypted offline. Restored archive into a new root, clean CLI login
and exact attachment bytes verified, with Workspace/MyAccounts absent. Same host,
not clean physical hardware. Native negative checks passed for cross-user item
denial, owner capability/origin/admin/signup protection, duplicate root ownership,
wrong recovery password, archive corruption/plaintext cleanup, login rate limiting
and spoofed forwarded IP rejection. Full native procedure/scripts/receipts remain
outside source at tooling/my-vault; fixture secrets must never be published.

Browser review found official web client also rejects HTTP on localhost. Added
native Go local HTTPS on gateway port+1, unique ECDSA localhost-only self-signed
leaf (CA=false, server-auth only), and explicit owner trust action for current
Windows user. Startup never changes trust. CLI trust-local supports service owner
use, service identities are refused. Owner tools stay separate local HTTP with
capability/Host/Origin. Local config rewrites HTTPS endpoints; root HTTP redirects
to HTTPS. Public client Origin now requires HTTPS. Trust is pending user approval
for this test host; do not install trust or bypass browser warnings without it.
Question has been asked asynchronously. Native local HTTPS+CLI decryption passed
using process-specific NODE_EXTRA_CA_CERTS without Windows trust changes. Browser
login/create flows, desktop/extension, clean-host and service/reboot remain open.

Local source now includes canonical default-port origin separation, reserved TLS
port in Workspace preflight, local certificate tests, README recovery screenshot,
full license files and security review. Screenshot docs/screenshots/recovery.png
is the actual running owner UI with a verified encrypted backup. No private paths
or credentials are displayed. Source is still uncommitted/unpublished.

Security audit found10 engine lock entries. Updated ONLY Cargo.lock rustls0.23.45
to0.23.46, no application-source/crypto change. Replacement lock is committed-to-be
under upstream/Cargo.lock, SHA0e7beeed7017145f5f5cba36f347685f4e17b396531c9143984832117e3cd38b,
and required by upstream.lock.json, native build script and runtime packager.
Rebuilding static engine (exec38992) started12:09 local, still compiling final
engine around12:28. Log static-engine-build.log; target C:cache/static-target.
Old receipt/binary still refer to original lock: MUST replace after new build,
inspect native dependencies/version/hash, regenerate runtime/rebuild/retest.
The remaining6 audit entries are rsa0.9.10 and Rocket's unused direct TLS stack
rustls0.21/webpki0.101. Config-specific assessment in SECURITY-REVIEW.md, not a
suppressed clean audit. ProductionEligible remains false; do not claim production
acceptance. Go govulncheck shows no imported/used vulnerability, only unused module
openpgp GO-2026-5932. docs/acceptance.json now records actual checks and open gates.

Corresponding source archives prepared under tooling/my-vault/release:
engine+vendored Rust106MB, complete OSS web17MB (commercial bitwarden_license
excluded), Go vendor32MB. verify-source-archives.py passed completeness (including
source data directories). Official SDK source commit3e060f8096b8a767c1f94b725b12badeee2c0777
downloaded sdk-source.tar.gz (rename into release). THIRD-PARTY-LICENSES.txt has
1106 original notice files (~5.8MB). Preserve full Go/Rust/SDK original notices.
No native fixtures/secrets belong in release assets. publish.py prepares/validates
GitHub draft uploads and hashes after exact source commit; not run yet.

Current owned processes: native-import MyVault PID70328, old TLS fixtures61912
(18916->18914) and28832(18926->18924); restored latest wrapper MyVault-local-tls.exe
PID42048 on18914 HTTP/18915 HTTPS, original source server stopped. Verify identities
before stopping. Root C:cache/native-restored, import native-import, original
native-original. No live C:/tmp/workspace changes. Final dist/MyVault.exe is old
payload; final TLS wrapper executable is tooling/MyVault-local-tls.exe, also old
engine until security rebuild completes. Do not overwrite running binaries.

Workspace tests/vet passed; further canonical-origin/preflight/docs edits need
final gofmt/tests. Its build now includes six My Vault guides plus three notices,
independent catalog app, icon/README/overview and org extraction. New org pin/hash
still absent. Full release25 not built. Existing updater intentionally preserves
installed app selection and does not automatically add new apps; fresh installer
will include13. Do not silently change this policy or upgrade live installation.

Resume: collect security engine build; preserve new engine/receipt; stop owned
fixtures as needed; regenerate payload and final launcher, final source tests;
retest native backup/restore/replacement and local HTTPS, browser if trust approved.
Complete licenses/source metadata/acceptance honestly, commit/push new repo and
preview release only with clear open production gates. Generate vault update
contract from13-app prospective catalog, pin published org ZIP/hash, build complete
Workspace installer25 and test disposable fresh install13apps. Commit/push/release
Workspace with source/provenance/checksums and no Actions. Preserve unrelated
untracked installed-preview22 handoff, existing live apps/backups and old releases.

## Final component build verified

Security engine completed18m29s, SHA5aa19bc46dda061e20a373ddde0f92cc0f94ae15f2f14b61e8fb7e00f793169f,
42,179,072bytes, system DLLs only. Final launcher dist/MyVault.exe is66,552,832bytes.
Rebuilding current source produced identical executable hash. Final tests/vet and
govulncheck passed. Final native local HTTPS, existing-data reopen, new encrypted
backup, fresh-root restore/new TLS certificate, clean CLI password/note/attachment
decryption and offline unlock all passed with final security-patched binary.
Final source and restored servers stopped; import fixture/TLS proxies also being
stopped. Temporary native data are synthetic; never release fixture credentials.
Runtime package manifest verifies282files and unchanged client JavaScript.

Source archives were fully checked for every required file (including originally
omitted source data directories), with restricted commercial web code excluded.
Third-party notices, full Go/Rust/GPL/AGPL originals and SDK corresponding source
are packaged. ProductionEligible remains false: browser certificate trust and
registration/sign-in, desktop/extension, clean physical host/service/reboot/MFA/
passkey/collections and independent security review are not claimed. The pending
local certificate question has no answer; do not modify Windows trust yourself.

My Vault source/public preview publication is next. Windows package must exclude
Go's obsolete MyVault.exe~ backup; it was generated while an old fixture held the
prior executable. Stop owned import first and remove that build-only file.
prepare-package.py builds ZIP/notices, prospective catalog and file map;
Workspace cmd/update-manifest generates release/workspace-update.json. The old
local Workspace dist catalog has11apps; prospective12 includes vault only and is
used solely for its scoped independent contract. Full build-org uses current pins
and must verify all13apps, including My Tube. No component or Workspace release
has yet been published for this task. Full Workspace25 build/test still remains.

## Component publication and suite verification

Published v0.1.0-preview.1 from1ff33c0e5cb33e817b3f741ddd97162c3fd042aa,
all GitHub asset sizes/SHA256 checked. Windows ZIP SHA
d0912e0f7780f4dbb4d4525480147f7b1079d1f45627d704a3aee7318064da1d,
57,952,329bytes. Standalone executable66,552,832bytes. No Actions.
The SDK source asset initially included44 upstream separately licensed paths.
Temporarily drafted the release, removed that directory from the SDK source
archive and replaced its source asset/checksums, then republished. Release body
discloses this source-only packaging correction; all binary assets unchanged.
Do not reintroduce restricted bitwarden_license code into any source archive.

Workspace full installer25 was built with verified org assets and passed fresh
isolated offline installation:13entries/11process health checks, exact My Vault
org binary, no MyAccounts dependency, anonymous launch instructions to separate
local HTTPS, recovery guides installed and prior autostart unchanged. Final
manager/catalog/embedded setup refreshed after the last localhost-route fix;
all packaged file bytes and vault's independent update contract verified.
All test supervisor/app/shared-service processes stopped; live suite untouched.
Workspace source42363fb, its release upload is underway at tooling/my-vault.

Windows trust is still unchanged: certificate approval question unanswered.
Browser/desktop/extension, second physical host, MFA/passkeys/collections,
service/reboot and independent security acceptance remain explicitly open.
Do not change productionEligible to true or call this a production release.
