# Contributing

## Package changes

Read [How to add a package](docs/how-to-add-a-package.md) before you open a package pull request.

1. Open an issue that identifies the upstream repository and the package owner.
2. Add or update one file in `catalogue/packages/`.
3. Attach evidence for the license, immutable release asset, publisher checksum, Knulli compatibility, install paths, preserved files, network use, and every write path. Upstream-managed experimental packages may use a pinned GitHub asset digest when no publisher checksum exists; record the API evidence and disclose this in the install warning. This does not provide independent publisher authentication.
4. Keep the entry at `candidate` while runtime or lifecycle safety is unresolved. Candidates can retain validated release metadata, but no compatibility or install definition.
5. Use `experimental` for a warned, authorized test after release, runtime and lifecycle review. One maintainer can complete this review. Use `installable` only after the full two-reviewer checklist passes. Device and display declarations are optional for experimental packages; missing exact device evidence does not block installation.
   An authorized `install.upstream_managed` package can use verified download/extraction while leaving runtime setup and updates to the app. It must stay experimental, disclose these risks, preserve user data, and declare cleanup checks for external hooks. An unverified license needs a visible warning and direct upstream download without redistribution. See [upstream-managed installation](docs/how-to-add-a-package.md#upstream-managed-installation).
6. Use `verified` only when a linked report shows a successful test on the exact device, firmware, architecture, and resolution in the manifest.

A README statement alone is not compatibility evidence. Do not add installers that pipe remote data to a shell or that need unrestricted writes. Do not add ROM-download sources.

## Code changes

Keep the installer independent from presentation code. New filesystem behavior needs a temporary-root integration test. New archive behavior needs an adversarial fixture where an unsafe implementation would produce a different result.

Run:

```sh
make check
make build
make catalogue
make catalogue-check
```

Generated build output is not committed. `make catalogue-check` is the local gate: it builds the index twice and requires the two builds to be byte-identical. A `git diff --exit-code -- build/catalog-index.json` cannot stand in for it, because `build/` is gitignored and git reports an ignored path as clean either way. Device CI signs `catalog-index.json` and compiles the matching public key into that build.

The [catalogue updates workflow](.github/workflows/catalogue-updates.yml) checks every package daily at 12:00 PM Singapore time (04:00 UTC; GitHub may delay scheduled runs). It prepares changed experimental metadata only after verifying the downloaded asset size, SHA-256, publisher checksum where required, and bounded archive extraction. Renamed or ambiguous assets, binary patches, embedded payloads, missing evidence, candidates, and higher technical statuses remain manual-review items. Upstream code is never executed. An API metadata error fails the run; blocked proposals remain visible in the report.

Changed proposals must pass `make check`, SDL race tests, coverage, catalogue determinism, and Staticcheck with both tag sets before the workflow drafts a PR with What, Why, and How sections. It skips duplicate PR creation while an automation proposal is open, and never overwrites that PR. No metadata change means no PR. Reports are available in the run summary and artifacts. The JSON report retains each inspected file's staged path (after `strip_components`), mapped install path, byte size, and SHA-256 without temporary host paths. The PR links to the run's artifacts; its Markdown report summarizes the inventory instead of embedding an unbounded file list. Merge, runtime review, and real-device validation remain manual. After a human merges to `main`, successful normal checks automatically publish an experimental prerelease through the existing release workflow. Full releases require a `v*` tag. Repository Settings → Actions → General must allow GitHub Actions to create pull requests. The job uses `GITHUB_TOKEN` with contents and pull-request write permissions; no additional secret is needed. PRs created with that token do not trigger other workflows, so validation runs before PR creation.

Local use stays read-only: `go run ./cmd/check-updates -output build/catalogue-update-report` writes JSON and Markdown reports without downloading assets or editing manifests. Add `-apply` only to prepare local proposals for human review. Apply rejects symlinked manifest paths (including directory components) and nonregular manifest files, and uses atomic replacement for writes. `GITHUB_TOKEN` is optional locally and increases the API rate limit; never put it in a report. The repository and open GitHub PRs are the automation's persistent state; no Amp schedule is required.

Comments must explain a design reason that the code cannot make clear by itself.
