# Sandbox runtime

The Rust runtime runs inside one sandbox task and owns child processes, command output, filesystem operations, and `/workspace`. It must not contain lifecycle, dashboard, or AWS application logic.
