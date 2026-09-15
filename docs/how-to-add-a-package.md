# How to add a package

## What

Add one reviewed manifest under `catalogue/packages/`. Give the package a stable reverse-domain ID and link its upstream project, license, maintainer, and evidence. A package stays `candidate` until every mandatory technical field is safe. Community approval does not make it installable or verified.

## Why

Package code runs with access to user files. A complete review protects ROMs, saves, configuration, storage, and the Knulli menu. Exact release metadata also makes an install repeatable and repairable.

## How

1. Open an issue with the project link, maintainer, purpose, target devices, and known risks.
2. Select an immutable versioned release asset. Do not use `latest`, a branch archive, or an unrestricted remote install script.
3. Record its exact compressed size, extracted size, SHA-256, supported ZIP or `tar.gz` format, license SPDX expression, and evidence URLs.
4. Inspect every archive entry. Reject absolute paths, traversal, links, devices, pipes, duplicate names, and undeclared launchers.
5. Record architecture and dynamic dependencies. Test firmware, device ID, and every resolution separately; a shared chip does not prove compatibility.
6. Declare the narrowest install destination and allowed `/userdata` writes. List user data and configuration in preserve paths. List each launcher or binary that needs execute mode and explain why.
7. Review runtime writes, network use, self-updaters, uninstall behavior, menu integration, and rollback. Updates must stay inside App Store checksum and transaction controls.
8. Add manifest, compatibility, archive, install, repeated-install, repair, uninstall, preserved-data, and rollback tests. Add GUI layout tests for each target resolution.
9. Run:

   ```sh
   gofmt -w .
   go vet ./...
   go vet -tags sdl ./...
   go test -race ./...
   go test -tags sdl ./...
   go run ./cmd/knulli-app catalogue -output build/catalog-index.json
   git diff --exit-code -- build/catalog-index.json
   ```

10. Open a pull request with the security review, archive inventory, exact commands, checksums, dependency output, screenshots at each target size, and this checklist.
11. Use `experimental` only for a clear, user-authorized hardware test after all mandatory release and path controls pass. Record tester, date, firmware, device, architecture, resolution, package version, and exact lifecycle results. Use `verified` only for the exact matrix proven on real hardware.

For an update, repeat the full release, archive, dependency, security, lifecycle, and device review. Never copy values from the weekly report without verification. For removal or deprecation, remove the catalogue and product references, regenerate the index, and retain only generic orphan-state handling for old manager state.
