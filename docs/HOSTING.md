# Dedicated HTTPS hosting

Keep the vault on a separate origin such as `https://vault.example.com`, rather
than sharing the browser origin of Drive, Notes or other Workspace apps. Those
apps must not be able to read the vault client's local browser storage. The
Workspace app tile is a link/launcher, not a replacement vault login.

The Rust engine binds to an ephemeral loopback port. My Vault's gateway binds to
the configured fixed loopback listener (8914 standalone; Workspace base+14).
Only that gateway is a reverse-proxy target. Never expose the engine, `/admin`,
diagnostics or the separate owner-management listener directly.

1. Establish a dedicated HTTPS hostname with a trusted certificate using a
   reverse proxy or Cloudflare tunnel. Route it to `http://127.0.0.1:8914` (or your
   configured fixed listener). Do not create a public HTTP listener.
2. In local owner tools save the dedicated public URL and restart My Vault.
3. Set that URL as the self-hosted server in compatible clients.
4. Verify HTTPS, allowed Host, login, sync and WebSocket notifications. Keep local
   direct access and independent encrypted backups available during an outage.

Example Cloudflare ingress fragment (the tunnel has its own protected credentials):

```yaml
- hostname: vault.example.com
  service: http://127.0.0.1:8914
```

Configure DNS/tunnel access in your domain account; a saved URL alone does not
provision DNS or certificates. The public gateway rejects remote plain-HTTP
configuration. Proxy headers do not authorize local owner tools or signup.

When the domain/proxy is lost, run the launcher with the same data root and use
the direct loopback vault link from local owner tools. Its API environment is
mapped to the loopback address without changing encrypted items or passwords.
If the entire server is lost, follow RECOVERY.md on a fresh computer.

## Local HTTPS

The HTTP gateway remains loopback-only for health and a trusted external proxy.
The following port serves local HTTPS (default 8915; Workspace app port + 1).
Owner tools remain on their separate capability-protected loopback HTTP port.
Use **Trust My Vault for this Windows account** explicitly before local browser
sign-in. It installs only a unique self-signed localhost/127.0.0.1 server leaf
with CA=false and no certificate-signing key usage. Startup never adds trust.
Its private key stays in the protected local-tls folder; it is excluded from
portable server backups. Recovery creates a fresh certificate on the new host.
The leaf expires after one year. Preserve old certificate files and remove the
old thumbprint from Current User/Trusted Root before explicitly provisioning a
replacement. Removing trust does not delete vault data. Do not trust this
localhost certificate on remote devices or bypass browser warnings.

When running as a Windows service, perform trust from your own login using
`MyVault.exe trust-local --data YOUR_DATA_FOLDER`. Trust is per Windows user;
services never add a certificate to a different user’s store.
