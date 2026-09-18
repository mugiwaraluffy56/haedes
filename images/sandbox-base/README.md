# Sandbox base image

This image is the pinned, non-root Linux base for the default haedes sandbox. It builds the Rust runtime from the repository and installs the planned development tools: Git, curl, wget, Bash, Python 3, pip, Node.js, npm, Go, Rust, GCC, G++, and Make.

The final image uses a pinned Debian digest, exposes only the internal runtime port `8080`, creates writable `/workspace` and `/tmp` directories for UID `10001`, and starts through `entrypoint.sh`. The entrypoint removes inherited AWS credential variables and refuses to start unless `HAEDES_RUNTIME_TOKEN` and `HAEDES_SANDBOX_ID` are set.
