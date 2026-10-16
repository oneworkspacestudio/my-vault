# Compatible clients and format boundaries

Use official clients from Bitwarden's website/stores. In the login environment
selector choose **Self-hosted** and enter the dedicated My Vault HTTPS URL. For
all current clients, including the browser and CLI, HTTPS is required even on
localhost. The local owner tools offer an explicit current-user trust action for
the unique localhost leaf certificate. CLI 2026.9.1 uses its bundled Node trust
store; set `NODE_EXTRA_CA_CERTS=YOUR_DATA_FOLDER/local-tls/localhost.pem` for
that CLI process. Configure publicly trusted HTTPS for native clients and
cross-device access; do not disable certificate verification.

| Client | Setup | Evidence policy |
| --- | --- | --- |
| Browser web vault | Open the dedicated vault address | Online interface only; not an offline client |
| Browser extension | Set self-hosted server URL before sign-in | Test sign-in, sync, lock and offline unlock for the exact version |
| Windows desktop | Set self-hosted URL; sign in and sync | Test offline unlock with retained logged-in app data |
| Official CLI | `bw config server URL`; login/unlock; sync | Use password/session input securely; avoid command-line secrets and output logs |
| Android/iOS | Future device acceptance; same self-hosted URL | Deferred by owner; no mobile release/acceptance claimed |

The server is Vaultwarden 1.37.4 and the compatible web vault 2026.7.0. Supported
features must be checked against the actual client version in acceptance.json.
Passkey storage, passkey sign-in, MFA passkeys and exporting passkeys are different
capabilities. The upstream compatible web build hides unsupported passkey sign-in;
My Vault does not re-enable it through cosmetic changes.

## Export/import is not a complete filesystem backup

- Prefer **password-protected encrypted JSON**, never an account-restricted file
  as the only recovery copy. A target independent account must successfully import.
- Personal exports need not contain organisation-owned items. Export organisation
  data with appropriate ownership/permissions and test separately.
- Attachments/Sends are not assumed included in JSON. Keep full server backups and
  separate attachment recovery checks.
- TOTP seeds, passkeys, SSH keys, collections, item types and custom fields vary by
  client/export format. CSV is lossy and plaintext; it is not a secure backup.
- MFA secrets/recovery codes for signing into the vault are not a vault-item export.
  Keep those independently.
- No automatic cloud/mobile push, external breach service or favicon fetching is
  configured by this package. Desktop/browser clients can sync normally.

The server enforces upstream login rate limits. The loopback gateway deliberately
ignores arbitrary forwarded client-IP headers, so users behind the same proxy may
share a rate-limit bucket. Configure dedicated trusted proxy handling only through
a reviewed change, never by trusting browser-provided IPs.
