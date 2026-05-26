# GIRL with OpenCode

GIRL integrates with OpenCode as a project-specific refactoring agent.

## Installation

```bash
# Copy GIRL agents to your project
cp -r opencode/agents/* .opencode/agents/

# Make sure girl is on PATH
go install ./cmd/girl/
```

## Agents

### `girl-planner`

Analyzes code and generates structured GRP plans without editing files.

Invoke with:
```txt
@girl-planner analyze src/components and generate a plan
```

### `girl-implementer`

Applies approved GRP plans step by step, verifying after each change.

Invoke with:
```txt
@girl-implementer apply the plan from .grp/plan.json
```

### `girl-reviewer`

Reviews refactored code for quality and behavior preservation.

Invoke with:
```txt
@girl-reviewer review the refactored component
```

## Workflow

1. Run `girl analyze <path>` to detect smells
2. Run `girl plan <path>` to generate a GRP plan
3. Review the plan with `@girl-planner`
4. Apply with `@girl-implementer`
5. Verify with `@girl-reviewer`

## Commands

| OpenCode | GIRL CLI Equivalent |
|----------|-------------------|
| `@girl-planner this component is too large` | `girl plan src/ --output markdown` |
| `@girl-planner find repeated patterns` | `girl analyze src/ --output text` |
| `apply the plan` | `girl verify . && implement` |
| `review the refactor` | `girl analyze --output markdown && girl verify` |
