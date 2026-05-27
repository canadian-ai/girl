# GRP-React Diagnostics v0.1

## react.large-component

| Field | Value |
|-------|-------|
| **Code** | `react.large-component` |
| **Title** | Component is too large |
| **Description** | Component exceeds 150 lines. Large components mix rendering logic, state, and effects. |
| **Severity** | `medium` |
| **Confidence** | `high` |
| **Recipes** | `react.split-large-component`, `react.extract-custom-hook` |
| **False positive risks** | Page-level layout components may legitimately be large. |

## react.too-many-hooks

| Field | Value |
|-------|-------|
| **Code** | `react.too-many-hooks` |
| **Title** | Too many hooks in a single component |
| **Description** | Component uses 6+ hooks. Consider extracting related hooks into custom hooks. |
| **Severity** | `medium` |
| **Confidence** | `high` |
| **Recipes** | `react.extract-custom-hook` |
| **False positive risks** | Complex forms may legitimately use many hooks. |

## react.too-many-state-vars

| Field | Value |
|-------|-------|
| **Code** | `react.too-many-state-vars` |
| **Title** | Too many state variables |
| **Description** | Component declares 5+ `useState` calls. Consider consolidating related state or using `useReducer`. |
| **Severity** | `low` |
| **Confidence** | `medium` |
| **Recipes** | `react.reduce-state-vars`, `react.consolidate-effects` |
| **False positive risks** | Forms with many independent fields may need separate state vars. |

## react.too-many-effects

| Field | Value |
|-------|-------|
| **Code** | `react.too-many-effects` |
| **Title** | Too many useEffect calls |
| **Description** | Component uses 4+ `useEffect` hooks. Effects that run together should be consolidated. |
| **Severity** | `medium` |
| **Confidence** | `high` |
| **Recipes** | `react.consolidate-effects` |
| **False positive risks** | Components subscribing to multiple independent data sources. |

## react.repeated-jsx

| Field | Value |
|-------|-------|
| **Code** | `react.repeated-jsx` |
| **Title** | Repeated JSX pattern |
| **Description** | Same JSX structure appears 3+ times. Extract into a sub-component. |
| **Severity** | `low` |
| **Confidence** | `medium` |
| **Recipes** | `react.extract-repeated-jsx` |
| **False positive risks** | Maps over static data may look like repeated JSX. |

## react.mixed-responsibilities

| Field | Value |
|-------|-------|
| **Code** | `react.mixed-responsibilities` |
| **Title** | Component mixes data fetching, UI, and business logic |
| **Description** | Component handles data loading, formatting, event handling, and rendering without separation. |
| **Severity** | `high` |
| **Confidence** | `medium` |
| **Recipes** | `react.split-large-component`, `react.extract-custom-hook` |
| **False positive risks** | Simple presentational components may intentionally own their data. |

## react.hardcoded-data

| Field | Value |
|-------|-------|
| **Code** | `react.hardcoded-data` |
| **Title** | Hardcoded data in component body |
| **Description** | Inline arrays, objects, or strings over 3 lines in the component body are recreated every render. |
| **Severity** | `low` |
| **Confidence** | `high` |
| **Recipes** | `react.extract-constants` |
| **False positive risks** | Truly constant lookup tables are acceptable. |

## react.missing-prop-types

| Field | Value |
|-------|-------|
| **Code** | `react.missing-prop-types` |
| **Title** | Component props are not typed or validated |
| **Description** | Component accepts props without TypeScript interface, PropTypes, or JSDoc. |
| **Severity** | `medium` |
| **Confidence** | `medium` |
| **Recipes** | `react.add-prop-types` |
| **False positive risks** | Wrapper components that pass all props through. |
