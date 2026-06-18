# GIRL Benchmarks and Proof Reports

GIRL benchmark reports summarize analyzer findings for a local repository clone.
They are designed to make refactoring risk and agent-handoff risk easy to share
in issues, pull requests, and screenshots.

## Benchmark a local clone

Clone a repository locally, then run `girl benchmark` against the checked-out
folder. For example, to benchmark Kubernetes Go code:

```bash
git clone https://github.com/kubernetes/kubernetes
girl benchmark kubernetes --lang go --output markdown
```

## Generate a proof report

Use `girl prove` when you want a higher-level health report with a 0-100 score
and suggested improvement categories. For example, to inspect a Next.js clone:

```bash
git clone https://github.com/vercel/next.js
girl prove next.js --lang ts --output markdown
```

## How to interpret reports

GIRL is not judging code quality universally. A diagnostic count is not a moral
score and does not replace project context, domain expertise, or maintainers'
judgment.

Instead, GIRL surfaces:

- refactoring risk, such as high-complexity functions, deep nesting, and large
  files or components;
- agent-handoff risk, where broad or tangled code may be harder for an AI coding
  agent to modify safely;
- repeatable evidence that can be compared across commits or local experiments.

The benchmark and proof commands operate on local paths only and reuse the same
analyzer pipeline as `girl analyze`.
