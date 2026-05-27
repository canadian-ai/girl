# GRP-React Verification v0.1

- Binding name: `GRP-React v0.1`
- Layers on GRP-TypeScript verification; React inherits all TypeScript verification detection.

## Additional detection

| Rule | Match | Verification |
|------|-------|-------------|
| React project | `react` or `next` in `package.json` dependencies | Extends TS verification |
| JSX/TSX files | `.tsx` / `.jsx` files present | `npx tsc --noEmit` |
| Test framework | `jest`, `vitest`, `@testing-library/react` in deps | `npm test` |

## Verification commands

React verification includes all TypeScript verification commands plus:

- `npx tsc --noEmit` (when `tsconfig.json` exists)
- `npm test` (when test framework detected)
- Component-specific lint rules when `eslint-plugin-react-hooks` is installed
