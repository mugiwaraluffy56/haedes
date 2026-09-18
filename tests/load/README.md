# Load tests

Command and lifecycle load scenarios belong here.

Run the bounded local/development scenario with:

```sh
node tests/load/run-load.mjs
```

It defaults to four virtual users, two concurrent commands per sandbox, and
two create/destroy rounds. Set `HAEDES_API_URL`, `HAEDES_API_KEY`, and the
`HAEDES_LOAD_*` bounds to target a local or development control plane. Use
`--dry-run` to print the bounded plan without making requests. The JSON output
is a local measurement, not a production performance claim.
