# Repeatable local Windows build

Dependencies are build-time only: Rust 1.99.0 (MSVC target), Visual Studio C++
tools/Windows SDK, native Strawberry Perl for bundled OpenSSL, Go 1.27.2, Python3
and Git. Download toolchains from their upstream publishers and verify digests.
Keep native build state in a named NTFS cache and large downloads on your build
drive. Do not use GitHub Actions or unverified third-party vault binaries.

1. Clone exact upstream Vaultwarden commit from upstream.lock.json and verify
   `git rev-parse HEAD`. Apply only the supplied reviewed upstream/Cargo.lock
   security dependency update (rustls 0.23.46), verifying its locked hash. Inspect published security
   advisories. Do not silently use master or a newer incompatible web client.
2. In the x64 MSVC environment build `cargo build --locked --release --features
   sqlite,vendored_openssl`. For a static CRT build use the explicit Windows target
   plus `RUSTFLAGS=-C target-feature=+crt-static`; inspect actual PE dependencies.
   Set `VW_VERSION=1.37.4`. Save a build receipt with commit, compiler, features,
   dependency-lock digest, executable digest and runtime dependency inspection.
3. Download the pinned `bw_web_builds` release archive and verify its locked SHA256.
   Preserve its exact corresponding web source and original notices.
4. Run `scripts/package-runtime.py --engine ENGINE_EXE --web WEB_VAULT_FOLDER
   --stage FRESH_STAGE --build-receipt RECEIPT_JSON`. The receipt binds the engine
   to its reviewed commit/hash. This script changes only index title, additive
   CSS, SVG and a recovery link; it checks every JavaScript file is unchanged.
5. Build `go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=VERSION"
   -o dist/MyVault.exe ./cmd/myvault`. Run the tests and native recovery acceptance
   in a dedicated disposable NTFS fixture. Never use a live vault/identity store.
6. Package launcher, licenses, exact corresponding source, acceptance matrix,
   `workspace-update.json`, notices/provenance, recovery/client/admin guides and
   SHA256SUMS.txt. No user data or private settings enter the source/package.

The normal launcher verifies embedded runtime bytes before extraction and again
on every start, and holds the engine executable against replacement. A build
receipt is provenance, not a security audit or a claim of byte-identical
reproducibility across arbitrary compiler versions.

Production eligibility requires the explicit independent recovery checks. A
successful build does not prove browser/desktop/offline/export interoperability.
