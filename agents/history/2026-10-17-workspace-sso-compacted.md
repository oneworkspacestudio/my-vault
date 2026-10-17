# Workspace sign-in and first-account handoff

## Request and confirmed causes

Owner reported My Vault showing a second login form after opening it from an
already authenticated Workspace. Read the prior RC2 handoff and reconciled clean
component Git state with the newly installed suite13-app catalog.

Three independent integration problems were confirmed during the requested repair:

1. Managed optional SSO explicitly disabled upstream SSO account provisioning.
   A first user was rejected and returned to login instead of encryption setup.
2. The installed owner's email was not verified in MyAccounts. Its standard JWT
   omits false email_verified, so upstream correctly rejected unknown status.
3. Client-config used display identifier `one-workspace`. First-account policy
   fetch then targeted that nonexistent organization, returned unauthorized and
   logged the client out. The pinned native engine requires its canonical SSO
   identifier `00000000-01DC-01DC-01DC-000000000000` for its dedicated enrollment
   and master-password-policy routes. This own integration error explains the
   observed bounce; do not blame the upstream report alone.

## Implementation

SSOConfig adds opt-in `provisionAccounts` (defaultfalse for standalone settings).
Managed Workspace writes true for its dedicated client, retaining explicit owner
SSO opt-out. Uses upstream JIT, OIDC/PKCE, verified-email checks and ordinary
client encryption initialization. Anonymous signup stays closed; independent
password login remains available. No custom cryptography, password collection,
session impersonation, unknown-email allowance or MFA bypass was introduced.

Public client-config and additive navigation JS now agree on the canonical native
SSO identifier. Existing once-per-Workspace-launch automatic navigation retains
callback/error loop protection and lets upstream reuse locked/unlocked sessions.
Removed the duplicate independent email form from the SSO view; explicit
independent-login URL retains it. Updated hints, README and SIGN-IN guide.

## Local deployment and evidence

Built native launchers locally; no Actions or test suite was added/run. Go builds,
JavaScript syntax and package digest checks passed. Packaging confirms the engine
and original compatible-client JavaScript are unchanged. Two local repair bundles
were applied with the supported stopped-state backup/update/restart transaction.
Both complete with verified backups, identical suite logical credentials,
storage/settings/app selection retained and local/public suite health200/ok.
First repair exposed the canonical-identifier error; the second fixes it.

Final local component version `0.2.0-rc.2+workspace-sso.2`; manager local version
`0.3.0-preview.31+vault-sso.1`. Artifacts/receipts/scripts outside Git under local
tooling/my-vault/seamless-sso; protected maintenance evidence stays in private data.
Do not rerun those one-use deployment scripts blindly or delete real snapshots.

Owner email verification was recorded only after the authenticated Gmail connector
profile exactly matched the installed owner email; no email challenge bypass or
unverified address was assumed. Only email_verified was updated through native
MyAccounts admin API; passwords were unchanged. Private evidence contains only the
matching-mailbox proof method and email digest. No MyAccounts source was changed.

Normal owner Chrome launch with `?workspace=1` now reuses the existing MyAccounts
browser session and reaches native `#/set-initial-password` with the canonical
identifier, without email entry or another identity login. Screenshot of blank
encryption setup is retained privately; Chrome tab585733441 marked handoff.
The owner must choose/submit the real master password themselves. No real master
password was entered, reset or read; password submission/encrypted-item acceptance
was not performed. This observation is a deployment sign-in handoff, not a fresh
isolated/native recovery acceptance. Historical RC2 acceptance remains historical.

## Publication and limits

No commit/push/release for this task; source changes and local builds are unreleased.
Public RC2/Workspace31 are unchanged and still contain the old integration. The
source distribution pin must be updated when publishing a component/suite release;
do not imply a public installer contains this local repair. Public updater can
offer the old same-source component pin; publish a compatible new release before
using it to replace the local repaired Vault. Production/client/recovery gates
remain as documented. New users still need a genuinely verified MyAccounts email;
the owner's connected-mailbox proof is not a general installer verification rule.
Local unique certificate, private root, independent recovery and existing tunnel
remain intact. Do not revive the old18935 synthetic browser signup fixture.

## Remote hosting and publication authorized

Owner now explicitly requests equivalent remote hosting and all changes pushed
and released. Targets are component v0.2.0-rc.3 and Workspace v0.3.0-preview.32,
built locally without Actions. The manager will allocate an unused dedicated
first-level hostname on its existing managed tunnel, preserve other DNS/routes,
retain the chosen address across updates and configure the exact native callback.
Component local recovery suppresses cross-origin SSO when its public hostname is
configured, retaining independent local login. No encryption/engine dependency
change; remote deployment and publication evidence will be appended separately.

## RC3 publication and managed remote deployment completed

Published [v0.2.0-rc.3](https://github.com/oneworkspacestudio/my-vault/releases/tag/v0.2.0-rc.3)
from732e782a8ae26f073c3d11701368a5adfaecea44. All26 release asset sizes/digests
verified; anonymous release metadata/Windows ZIP download respond200. Local build
only, Actions disabled. Workspace preview32 pins this exact org ZIP/hash and is
published with all13 apps. Deployment retains original engine/client JS, encrypted
state, certificates and logical suite credentials. No password change/reset.

The manager provisions a dedicated permanent HTTPS hostname on the existing
managed tunnel, preserving all earlier DNS/ingress. First owner deployment rolled
back on stale local upstream DNS; after cache expiry a fresh backed-up coordinated
deployment completed. Fresh encrypted native Vault backup verified; cold Accounts
and settings copied while stopped, prior full snapshots retained. No new native
full-suite transaction, clean-host or recovery acceptance claimed.

Owner Chrome launch from signed-in Workspace now reaches remote native #/lock
without another account login or certificate warning. Master password is still
required to decrypt; it was not entered/read. Owner tab left as handoff and blank
unlock screenshot only in protected private maintenance. No item/save/unlock,
client/recovery/service/reboot or automated tests added/run this task. Historical
RC2 acceptance remains historical; productionEligible remains false. Stable origin
and callback retained across updates; independent local HTTPS recovery preserved.

Receipts outside Git: tooling/my-vault/release-rc3-build and Workspace release32
folder. A documentation-only completion commit follows release source without
retagging or replacing assets. No private evidence/screenshots/credentials in Git.
