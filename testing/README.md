# Testing Guide

To run integration tests:

```bash
make integration
make integration TEST=create   # runs event creation flow
make integration TEST=register # runs registration flow
```

Must install K6 to run load tests.
To run load tests:

```bash
make load
```

These commands spin up Docker Compose (dev base + integration or load overrides). Containers stay running so you can inspect logs or state before teardown.

```bash
make teardown                     # integration + load stacks
make teardown-integration          # integration stack only
make teardown-load                 # load stack only
```