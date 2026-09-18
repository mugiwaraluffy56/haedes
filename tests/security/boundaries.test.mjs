import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const read = (path) => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8');
const baseImage = read('images/sandbox-base/Dockerfile');
const entrypoint = read('images/sandbox-base/entrypoint.sh');
const sandboxTask = read('infrastructure/terraform/modules/sandbox/main.tf');
const sandboxSecurityGroup = read('infrastructure/terraform/modules/control-plane/main.tf');
const sandboxGroup = sandboxSecurityGroup.match(/resource "aws_security_group" "sandbox"[\s\S]*?resource "aws_ecs_task_definition"/)?.[0] ?? '';
const sandboxIngress = sandboxGroup.match(/ingress \{[\s\S]*?\n  \}/)?.[0] ?? '';
const ecsProvisioner = read('services/control-plane/internal/aws/ecs/provisioner.go');
const runtimeRunner = read('services/sandbox-runtime/src/process/runner.rs');
const runtimePaths = read('services/sandbox-runtime/src/filesystem/paths.rs');
const runtimeArchive = read('services/sandbox-runtime/src/snapshot/archive.rs');

assert.match(baseImage, /USER sandbox/);
assert.doesNotMatch(baseImage, /--privileged|docker\.sock|host\.?path/i);
assert.match(entrypoint, /unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE/);
assert.match(entrypoint, /HAEDES_RUNTIME_TOKEN/);
assert.match(entrypoint, /HAEDES_SANDBOX_ID/);

assert.match(sandboxTask, /user\s*=\s*"10001"/);
assert.match(sandboxTask, /readonlyRootFilesystem\s*=\s*true/);
assert.doesNotMatch(sandboxTask, /task_role_arn/);
assert.doesNotMatch(sandboxTask, /hostPath|docker\.sock|privileged/);
assert.match(sandboxGroup, /Control plane to sandbox runtime/);
assert.match(sandboxGroup, /security_groups\s*=\s*\[aws_security_group\.service\.id\]/);
assert.doesNotMatch(sandboxIngress, /cidr_blocks\s*=\s*\["0\.0\.0\.0\/0"\]/);

assert.match(ecsProvisioner, /ErrRuntimeCredentialOverride/);
assert.match(ecsProvisioner, /AssignPublicIpDisabled/);
assert.match(runtimeRunner, /\.env_clear\(\)/);
assert.match(runtimeRunner, /process_group\(0\)/);
assert.match(runtimeRunner, /killpg/);
assert.match(runtimePaths, /contains_percent_escape/);
assert.match(runtimePaths, /UnsafeSymlink/);
assert.match(runtimeArchive, /MAX_ARCHIVE_BYTES/);
assert.match(runtimeArchive, /validate_symlink/);
assert.match(runtimeArchive, /validate_archive_path/);

console.log('security boundary checks passed');
