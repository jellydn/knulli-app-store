# ADR 0009: Verify managed content from a recorded signature

- Status: accepted
- Date: 2026-09-16

## Context

Every catalogue load asked the installer for a status per package, and every status read and hashed every managed file. On a device that means re-reading several megabytes from the SD card to answer a question whose answer only changes when the files change. The load runs at launch and again after every operation, so the cost repeats without new evidence.

## Decision

Record the size and modification time of each file the installer verifies, next to the hash it verified. A health check hashes a destination file only when its size or modification time no longer matches that signature. Record the signature only where the recorded hash is the digest of that destination file, so a preserved file, an unmanaged adopted file, or a file whose reviewed hash differs records none. State written before this signature existed has no modification time and keeps full hashing until its next operation rewrites it. Log how many files each check read, so a load's cost is visible in diagnostics.

## Consequences

### Positive

- A load that follows an install, update, repair, adopt, or uninstall reads no content, because the operation already verified those exact bytes.
- A load hashes only the files whose metadata moved.
- The rule depends on metadata every supported filesystem reports, including ones that normalize permissions.

### Negative

- Content verification trusts size and modification time. A change that preserves both is not detected until the destination metadata moves.
- A file whose timestamp moved without a content change is read again on every load until its next operation rewrites the state.
- An adopted file that differs from the reviewed release records no signature, so it keeps being hashed on every load.
