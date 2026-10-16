# Sign-in and unlock

My Vault uses the established compatible-client encryption model. MyAccounts
identifies you; the vault client still needs its encryption key to unlock.
An unlocked running client keeps working until its ordinary lock policy applies.
Refreshing/reopening the web app can require unlock. No master password or
decryption key is copied from MyAccounts, Workspace or another browser origin.

## With One Workspace

The updated Workspace manager registers a dedicated `workspace-vault` application
in its own MyAccounts instance and saves its settings in the protected vault data
folder. Its distinct callback is:
`https://YOUR-VAULT/identity/connect/oidc-signin`.

1. Create the independent vault account with its own master password first.
2. Verify the matching email address in MyAccounts. Unverified email is refused.
3. Open My Vault from Workspace. When sign-in is needed, upstream OIDC reuses an
   existing MyAccounts browser session. Otherwise sign in there.
4. Unlock the vault when prompted, separately from account authentication.

No passwordless first-time vault account is silently created through SSO. The
pinned web version has a reported first-account SSO onboarding issue; this
integration requires an already-created independent account. Existing-account
linking and repeat SSO were exercised with native MyAccounts and the compatible
browser client in an isolated Windows installation.

## Independent access

Use **Use independent vault sign-in**, or open the vault with
`?login=independent#/login`. Its ordinary email/password controls remain available.
This works with MyAccounts stopped. An SSO association does not remove password
login. Compatible clients use the same independent server URL.

The local owner's **Use independent sign-in only** button saves an opt-out that
Workspace preserves on future starts. Advanced standalone settings accept your
MyAccounts issuer and dedicated client credentials. Remote issuers require trusted
HTTPS; HTTP identity discovery is allowed only on loopback.

Client secrets live outside the executable in protected NTFS `myvault-sso.json`.
They are excluded from independent recovery archives. After restoring a backup,
test direct login before reconnecting an identity provider.

Workspace sign-out does not automatically lock the separate vault origin. Use
the vault's **Lock now** / **Log out** and its session settings on shared PCs.
Provider-only sign-in uses upstream vault sessions after authentication, allowing
access during an identity outage; revoke vault sessions separately when removing
access. No automatic unlocking or custom cryptography is implemented.

References: [upstream OIDC configuration](https://github.com/dani-garcia/vaultwarden/wiki/Enabling-SSO-support-using-OpenId-Connect),
[pinned web first-account report](https://github.com/dani-garcia/vaultwarden/issues/7848).
