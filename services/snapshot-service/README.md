# Snapshot service

The Go snapshot service owns workspace archive coordination and the object-store boundary. Runtime archive behavior remains owned by the Rust runtime.

The service verifies the runtime-provided byte count and SHA-256 digest while streaming each archive into a private, AES-256-encrypted object key at `sandboxes/{sandboxId}/snapshots/{snapshotId}.tar.zst`. Metadata is published only after verification. Restore rejects unknown, unavailable, or expired snapshots and verifies the stored object before handing it back to the runtime. Retention treats missing objects as already deleted and transitions expired metadata to `deleted` idempotently.
