# My Vault

Use repository-history and maintain agents/history after material work. This app
wraps pinned Vaultwarden application source and its compatible web client. The
reviewed dependency-only security lock is retained in upstream/Cargo.lock. Do not change
vault cryptography or substitute Workspace/MyAccounts authentication for vault
authentication. No secrets or personal vault data in source, logs or fixtures.

Build locally without GitHub Actions. Use I:/Projects/tooling/my-vault for large
artifacts and C:/Projects/.cache/my-vault-build for necessary NTFS build/fixtures.
Never use the owner's running Workspace, identity or vault as a test fixture.
Record actual client/platform acceptance; do not label pending checks as passed.
Independent recovery is a production release gate. Preserve data on uninstall.
