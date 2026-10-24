# ADR 0001: Build a transactional installer before the UI

- Status: accepted
- Date: 2026-09-15

## Context

The product needs a controller-native UI, but package installation changes persistent user files. UI work cannot establish whether those changes are safe, recoverable, or portable.

## Decision

Build a standard-library Go installer and CLI first. Keep manifests declarative, stage archives, constrain writes, and use reversible transactions. Keep SDL2 optional and outside the installer dependency graph.

## Consequences

Safety behavior can run in temporary filesystems and on development hosts. The recovery CLI can be statically cross-compiled. The visible device experience arrives later, after hardware facts are available. The first milestone is useful for catalogue review and integration testing but is not yet a consumer app store.
