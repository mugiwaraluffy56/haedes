# Authentication bug fixture

This repository is intentionally red on its first test run. The
`authenticate` function compares the username with the password instead of
looking up the expected password for the known user.

The deterministic fix is to compare the supplied password with the value in
the `users` table. The second test already protects the incorrect-password
case. Coding-agent examples should observe the failure, inspect the source,
make the smallest edit, and run the tests again.
