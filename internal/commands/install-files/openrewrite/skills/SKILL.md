---
name: openrewrite
description: OpenRewrite recipe generation from GIRL diagnostics. GIRL detects refactoring opportunities and exports OpenRewrite-compatible YAML recipes for automated transformation of Java/JVM code.
---

# OpenRewrite + GIRL

GIRL generates OpenRewrite YAML recipes from refactoring diagnostics.

## How It Works

1. `girl analyze <path> --lang java` detects refactoring opportunities
2. `girl plan --recipe openrewrite.export-yaml-recipe` generates an OpenRewrite YAML recipe
3. Apply the recipe with `mvn rewrite:run` or `gradle rewriteRun`

## Generated Recipe Format

```yaml
---
type: specs.openrewrite.org/v1beta/recipe
name: dev.refactor.GirlGeneratedRecipe
displayName: GIRL Generated Refactoring
description: Auto-generated from GIRL analysis
recipeList:
  - org.openrewrite.java.ChangeMethodName:
      methodPattern: com.example.OldClass oldMethod
      newMethodName: newMethod
```

## Diagnostics

- `openrewrite.refactor-opportunity` — Detected code pattern that maps to OpenRewrite recipes
- `openrewrite.export-yaml` — Export diagnostics as OpenRewrite YAML recipe format
