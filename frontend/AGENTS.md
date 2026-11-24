# Frontend rules for AI coding agents

These rules add to the root [`AGENTS.md`](../AGENTS.md) for everything under `frontend/`. If a rule here is wrong, change this file in a PR, do not work around it.

## Layers (Feature-Sliced Design)

```
src/
  app/        router, providers, layout, global.css
  pages/      one folder per route, composes widgets and features
  widgets/    big page blocks (table, sidebar, header)
  features/   one user action each: create-product, archive-product, filter-products
  entities/   business objects: data hooks, types, small UI (product, category)
  shared/     api, ui, lib, config, providers; knows nothing about the business
```

1. A layer imports only from layers below it: `app → pages → widgets → features → entities → shared`. Never sideways: a feature does not import another feature, an entity does not import another entity. Code two slices need moves one layer down.
2. A slice is imported only through its `index.ts` (public API): `@/features/product/edit-product`, never `@/features/product/edit-product/ui/modal`. Inside a slice, use relative imports (`../model/modal.store`).
3. Segments inside a slice: `ui/` for components, `model/` for state, hooks, schemas and types, `lib/` for helpers. A file holds one thing.

## Features are small

- One feature is one user action. Editing, archiving and activating a product are three features, not one `product-actions`.
- A feature owns its UI (button, menu item, dialog), its store and its mutation. The page mounts its dialogs; a widget places its triggers.
- When a component grows past one job, split it.

## API

- Every request goes through the client that orval generates from `src/shared/api/schema.json` (`npm run api:download && npm run api:generate`). No `axios`, `fetch` or hand-written URLs outside `src/shared/api`.
- Do not edit `src/shared/api/generated` or `src/shared/api/model`: regenerate them.
- Import from `@/shared/api` only: the client factories (`getProducts()`), the types and the error helpers (`parseApiError`, `getApiErrorMessage`, `getErrorStatus`).
- Server state lives in TanStack Query. Query keys come from the entity (`productKeys`). After a mutation, invalidate the entity's keys.

## UI

- Build screens only from components in `src/shared/ui` (shadcn on Base UI). No raw `<button>`, `<input>`, `<select>`, `<table>` or hand-made badges in features, widgets or pages. A missing component is added to `shared/ui` first (`npx shadcn add <name>`), then used.
- Icons come from `iconsax-reactjs`. Pick the icon that says what the action does (`Edit2` for edit, `ArchiveBox` for archive, `SearchNormal1` for search). The exception is primitive shapes iconsax only has in a box or circle: a plain cross, dots, a check mark and thin chevrons. Those are hand-drawn SVGs in `src/shared/ui/icons.tsx` (`CloseIcon`, `MoreHorizontalIcon`, `MoreVerticalIcon`, `CheckIcon`, `Chevron{Up,Down,Left,Right}Icon`), on iconsax's grid (24×24, stroke 1.5, `currentColor`); a new primitive goes there too. No other icon library, including inside `shared/ui`. `npx shadcn add` writes lucide imports (components.json cannot name iconsax): swap them for iconsax in the same change; `lucide-react` is not a dependency.
- Colours come only from theme tokens in `src/app/global.css` (`bg-primary`, `text-muted-foreground`, `bg-success/10`, `text-warning`). No palette classes (`bg-red-100`, `text-emerald-800`, `bg-white/10`), no hex, rgb or oklch in components. A new colour is a new token in `global.css` for both light and dark.
- No `style={...}` and no CSS in components. Styling is Tailwind classes; anything Tailwind cannot express goes into `global.css` as a token or an `@utility`.
- Every screen has loading, empty and error states. An empty state (`Empty`) fills all the height the page has left (`flex-1`), it never sits as a small box at the top.
- Text the seller sees is Russian. API errors are shown through `getApiErrorMessage`, which maps backend codes to Russian.

## Imports and paths

- Always the `@/` alias across slices and layers. Relative imports only inside one slice, and never more than one level up (`../`). No `../../`.
- Keep paths short: `@/shared/api`, `@/shared/ui`, `@/entities/product`. A path that reaches into a slice's internals (`@/shared/api/http/error`, `@/entities/product/model/types`) means its `index.ts` is missing an export.
- Keep folder depth to `layer/[group/]slice/segment/file`. Do not nest deeper.

## Checks

`npm run lint && npm run build` must pass before a commit. For UI changes, check the screen in the browser: loading, empty, error and success.
