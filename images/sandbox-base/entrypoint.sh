#!/usr/bin/env bash
set -euo pipefail

# Sandbox tasks never receive cloud credentials. Clear inherited values before
# starting the runtime, even when a caller accidentally passes them through.
unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE \
  AWS_DEFAULT_REGION AWS_REGION AWS_CONFIG_FILE AWS_SHARED_CREDENTIALS_FILE

: "${HAEDES_RUNTIME_TOKEN:?HAEDES_RUNTIME_TOKEN is required}"
: "${HAEDES_SANDBOX_ID:?HAEDES_SANDBOX_ID is required}"

exec /usr/local/bin/haedes-runtime "$@"
