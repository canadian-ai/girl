# Python Verification

Python analysis requires Python 3 locally and does not execute inspected modules.

For a Python repository, configure appropriate verification commands in `.girl/config.yaml`, for example:

```yaml
verify:
  typecheck:
    - python3 -m compileall -q src
  lint:
    - ruff check .
  test:
    - python3 -m pytest -q
```

Commands are examples, not automatically assumed to be installed. Use the repository's own environment manager (`uv`, Poetry, pip, or Hatch) and native commands. Do not run verification on untrusted code outside a sandbox.
