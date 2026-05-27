# GRP-React Recipes v0.1

## react.split-large-component

| Field | Value |
|-------|-------|
| **Description** | Break a large component into smaller sub-components |
| **When to use** | Component exceeds 150 lines or handles 3+ UI regions |
| **Diagnostic mapping** | `react.large-component`, `react.mixed-responsibilities` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## react.extract-repeated-jsx

| Field | Value |
|-------|-------|
| **Description** | Extract repeated JSX into a named sub-component |
| **When to use** | Same JSX pattern appears 3+ times |
| **Diagnostic mapping** | `react.repeated-jsx` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |

## react.extract-custom-hook

| Field | Value |
|-------|-------|
| **Description** | Move related hook calls (state, effects, refs) into a custom hook |
| **When to use** | Component has 6+ hooks or mixes data fetching with UI state |
| **Diagnostic mapping** | `react.too-many-hooks`, `react.mixed-responsibilities` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## react.reduce-state-vars

| Field | Value |
|-------|-------|
| **Description** | Consolidate related state variables into a reducer or derived state |
| **When to use** | 5+ useState calls where some state is derived from others |
| **Diagnostic mapping** | `react.too-many-state-vars` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## react.consolidate-effects

| Field | Value |
|-------|-------|
| **Description** | Merge related useEffect calls that run on the same dependencies |
| **When to use** | 4+ useEffect hooks with overlapping deps |
| **Diagnostic mapping** | `react.too-many-effects` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## react.add-prop-types

| Field | Value |
|-------|-------|
| **Description** | Add TypeScript interface or PropTypes for component props |
| **When to use** | Component accepts props without type validation |
| **Diagnostic mapping** | `react.missing-prop-types` |
| **Risk level** | Low |
| **Verification** | `npx tsc --noEmit` |

## react.extract-constants

| Field | Value |
|-------|-------|
| **Description** | Move hardcoded data outside the component to avoid re-creation on every render |
| **When to use** | Inline arrays, objects, or large strings in component body |
| **Diagnostic mapping** | `react.hardcoded-data` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |
