# GRP-Python Binding

- Binding: **GRP-Python v0.1**
- Namespace: `py.*`
- Maturity: **Experimental**
- Parser: Python 3 standard library `ast` (requires `python3` on PATH)
- Privacy: local parsing only; GIRL does not import or execute target files.

## Usage

```sh
girl analyze . --lang python -o json
girl plan . --lang python --output grp-json
girl check --all -o json
```

GIRL detects Python from `.py` paths, `pyproject.toml`, `setup.py`, `requirements.txt`, or `Pipfile`. A mixed-language repository is included in `girl check`. Virtual environments, caches, generated files, and common excluded folders are skipped.

See [diagnostics](diagnostics.md), [recipes](recipes.md), and [verification](verification.md).

Known limitation: Python syntax parsing relies on the locally installed Python version, and Python is not yet integrated with the independent TypeScript complexity ratchet.
