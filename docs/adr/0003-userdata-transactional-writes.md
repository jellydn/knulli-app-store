# ADR 0003: Constrain package writes to /userdata with transactional rollback

- Status: accepted
- Date: 2026-09-15

## Context

Package operations can damage ROMs, saves, configuration, and EmulationStation menus. A handheld install also faces power loss, shared `gamelist.xml` files, and filesystems that normalize permissions. The installer must be able to prove every destination and reverse a failed step.

## Decision

Allow writes only below `/userdata`. Resolve every destination through `internal/safefs`, reject symlink path components, and stage archives before any destination mutation. Record each change in a transaction, write installed state last, and restore snapshots in reverse order on failure. Preserve declared configuration during repair, update, and uninstall. Own a menu entry only when the installer created an unchanged exact launch path. After a committed menu change, ask Knulli's loopback `/reloadgames` endpoint to queue a refresh; if that request is not accepted, report restart required and leave the operation committed.

Health checks compare immutable managed files with the mode observed after copy, not only the requested staging mode. Runtime files outside the release inventory stay unmanaged.

## Consequences

### Positive

- Failed installs restore previous files instead of leaving a partial tree.
- Tests can exercise the full lifecycle under `-root` on a temporary filesystem.
- Shared or modified menu entries are not deleted.

### Negative

- Power loss is not journal-recovered across process restarts.
- Empty directories can remain after rollback or uninstall.
- Path checks do not defend against a hostile local process that races a checked directory into a symlink.
- Root can still change files below approved `/userdata` paths.
