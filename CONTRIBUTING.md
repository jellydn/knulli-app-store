# Contributing

## Package changes

Read [How to add a package](docs/how-to-add-a-package.md) before you open a package pull request.

1. Open an issue that identifies the upstream repository and the package owner.
2. Add or update one file in `catalogue/packages/`.
3. Attach evidence for the license, immutable release asset, publisher checksum, Knulli compatibility, install paths, preserved files, network use, and every write path.
4. Keep the entry at `candidate` until all metadata is independently reviewed.
5. Use `installable` only after the review checklist passes.
6. Use `verified` only when a linked report shows a successful test on the exact device, firmware, architecture, and resolution in the manifest.

A README statement alone is not compatibility evidence. Do not add installers that pipe remote data to a shell or that need unrestricted writes. Do not add ROM-download sources.

## Code changes

Keep the installer independent from presentation code. New filesystem behavior needs a temporary-root integration test. New archive behavior needs an adversarial fixture where an unsafe implementation would produce a different result.

Run:

```sh
make check
make build
make catalogue
git diff --exit-code -- build/catalog-index.json
```

Generated build output is not committed. Review the generated index locally and sign it only in trusted release automation.

The weekly release checker has read-only repository access. It downloads GitHub release metadata only and uploads a review report. Run it locally with `go run ./cmd/check-updates -output build/catalogue-update-report`; it writes a `.json` and a `.md` report for manual review and never edits a manifest. It does not download assets, edit manifests, approve updates, open issues, or merge changes. `GITHUB_TOKEN` is optional for local use and increases the GitHub API rate limit; never put it in a report.

Comments must explain a design reason that the code cannot make clear by itself.
