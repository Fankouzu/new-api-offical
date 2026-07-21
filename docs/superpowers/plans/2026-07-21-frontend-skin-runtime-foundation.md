# Frontend Skin Runtime Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a build-time frontend skin runtime that preserves the current default UI, lets a selected skin override approved public pages, and generates skin-only TanStack routes without affecting authentication, setup, error, or authenticated routes.

**Architecture:** `web/default` remains the only SPA host. A generated active-skin module selects one runtime manifest per build, existing public route files delegate through `SkinPage`, and a Bun generator creates TanStack file-route proxies for skin-only routes. The first delivery includes a default adapter and a custom scaffold that falls back to default pages; visual redesigns and public-domain extraction are separate follow-up plans.

**Tech Stack:** React 19, TypeScript 5.9, TanStack Router, Rsbuild 2, Bun, Tailwind CSS 4, Base UI/shadcn source components, Node test runner through `tsx --test` or `bun test` where sandbox IPC prevents `tsx`.

---

## Scope Boundary

This plan implements the approved architecture foundation only:

- build-time skin selection;
- typed runtime and build manifest contracts;
- default page compatibility;
- approved public-page overrides;
- skin-only route generation and collision validation;
- route-level style and portal boundaries;
- navigation contributions;
- default/custom build verification.

It does not design or implement the final custom homepage, model marketplace, rankings, about, or legal-page visuals. Those require page-specific visual specifications and separate plans. The custom scaffold deliberately falls back to the default implementation for every existing public page.

## File Structure

### New runtime files

- `web/default/src/skins/runtime/contracts.ts`: runtime manifest, page, shell, route, and navigation types.
- `web/default/src/skins/runtime/build-contracts.ts`: codegen-safe build manifest types without React imports.
- `web/default/src/skins/runtime/route-policy.ts`: overridable pages, host-owned paths, and forbidden prefixes.
- `web/default/src/skins/runtime/resolve-page.ts`: deterministic active/default page resolution.
- `web/default/src/skins/runtime/skin-page.tsx`: renders an approved public page through the active skin.
- `web/default/src/skins/runtime/skin-boundary.tsx`: scoped root plus route-lifetime body marker cleanup.
- `web/default/src/skins/runtime/skin-portal.tsx`: portal container context for skin-owned overlays.
- `web/default/src/skins/runtime/skin-route.tsx`: resolves generated skin-only route IDs.
- `web/default/src/skins/runtime/navigation.ts`: merges skin navigation contributions with host navigation.
- `web/default/src/skins/runtime/active-skin.gen.ts`: generated runtime import; default content is committed.
- `web/default/src/skins/runtime/active-skin-build.gen.ts`: generated build metadata; default content is committed.

### New skin packages

- `web/default/src/skins/default/build-manifest.ts`: default build metadata with no new routes.
- `web/default/src/skins/default/manifest.ts`: adapters to current public feature pages.
- `web/default/src/skins/custom/build-manifest.ts`: custom scaffold build metadata.
- `web/default/src/skins/custom/manifest.ts`: custom scaffold with shell and default-page fallback.
- `web/default/src/skins/custom/shell/custom-public-shell.tsx`: minimal scoped shell used only by custom-owned pages.
- `web/default/src/skins/custom/styles/index.css`: scoped token example under `[data-skin='custom']`.

### New generator files

- `web/default/scripts/skin-route-utils.ts`: pure path validation and route-file rendering functions.
- `web/default/scripts/generate-skin.ts`: selects the skin, writes active modules, and regenerates route proxies.
- `web/default/scripts/skin-route-utils.test.ts`: generator unit tests.
- `web/default/src/routes/(skin-generated)/.gitignore`: preserves the generated directory while ignoring route proxies.

### Existing files modified once

- `web/default/package.json`: generation, build, typecheck, and dual-build scripts.
- `web/default/rsbuild.config.ts`: run generation before TanStack route discovery in dev/build.
- `web/default/knip.config.ts`: ignore generated skin modules and route proxies.
- `web/default/src/routes/index.tsx`: delegate `home`.
- `web/default/src/routes/pricing/index.tsx`: retain search schema and delegate `pricing`.
- `web/default/src/routes/pricing/$modelId/index.tsx`: retain params/search schema and delegate `modelDetails`.
- `web/default/src/routes/compare/ai-api-pricing/index.tsx`: delegate `pricing`.
- `web/default/src/routes/rankings/index.tsx`: retain search schema and delegate `rankings`.
- `web/default/src/routes/about/index.tsx`: delegate `about`.
- `web/default/src/routes/privacy-policy.tsx`: delegate `privacyPolicy`.
- `web/default/src/routes/user-agreement.tsx`: delegate `userAgreement`.
- `web/default/src/hooks/use-top-nav-links.ts`: merge validated skin navigation contributions.
- `web/default/src/hooks/use-top-nav-links.test.ts`: preserve host rules and cover skin links.

## Task 1: Lock the Route and Page Policy

**Files:**
- Create: `web/default/src/skins/runtime/contracts.ts`
- Create: `web/default/src/skins/runtime/build-contracts.ts`
- Create: `web/default/src/skins/runtime/route-policy.ts`
- Create: `web/default/src/skins/runtime/route-policy.test.ts`

- [ ] **Step 1: Write the failing route-policy tests**

```ts
import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  assertSkinRouteAllowed,
  getPublicPagePath,
} from './route-policy'

test('maps every overridable page key to its host path', () => {
  assert.equal(getPublicPagePath('home'), '/')
  assert.equal(getPublicPagePath('pricing'), '/pricing')
  assert.equal(getPublicPagePath('modelDetails'), '/pricing/$modelId')
  assert.equal(getPublicPagePath('rankings'), '/rankings')
  assert.equal(getPublicPagePath('about'), '/about')
  assert.equal(getPublicPagePath('privacyPolicy'), '/privacy-policy')
  assert.equal(getPublicPagePath('userAgreement'), '/user-agreement')
})

test('rejects host-owned and protected skin-only paths', () => {
  for (const path of ['/', '/pricing', '/sign-in', '/setup', '/500', '/dashboard']) {
    assert.throws(() => assertSkinRouteAllowed(path))
  }
})

test('accepts a new public path with TanStack parameters', () => {
  assert.doesNotThrow(() => assertSkinRouteAllowed('/guides/$slug'))
})
```

- [ ] **Step 2: Run the tests and verify the policy does not exist yet**

Run: `cd web/default && bun test src/skins/runtime/route-policy.test.ts`

Expected: FAIL because `route-policy.ts` and its exports do not exist.

- [ ] **Step 3: Add the minimal contracts**

```ts
// src/skins/runtime/build-contracts.ts
export type SkinRouteBuildDefinition = {
  id: string
  path: `/${string}`
  componentImport: string
  navigation?: {
    labelKey: string
    position: 'header' | 'footer'
    order?: number
  }
}

export type SkinBuildManifest = {
  id: string
  routes: readonly SkinRouteBuildDefinition[]
}
```

```ts
// src/skins/runtime/contracts.ts
import type { ComponentType, LazyExoticComponent, ReactNode } from 'react'
import type { SkinBuildManifest } from './build-contracts'

export type PublicPageKey =
  | 'home'
  | 'pricing'
  | 'modelDetails'
  | 'rankings'
  | 'about'
  | 'privacyPolicy'
  | 'userAgreement'

export type SkinComponent = ComponentType | LazyExoticComponent<ComponentType>

export type SkinPageDefinition = {
  component: SkinComponent
  shell: 'self' | 'skin'
}

export type SkinRuntimeRoute = {
  id: string
  component: SkinComponent
}

export type ThemeManifest = {
  id: string
  build: SkinBuildManifest
  pages: Partial<Record<PublicPageKey, SkinPageDefinition>>
  routes: readonly SkinRuntimeRoute[]
  shell?: ComponentType<{ children: ReactNode }>
}
```

- [ ] **Step 4: Implement explicit route policy**

```ts
// src/skins/runtime/route-policy.ts
import type { PublicPageKey } from './contracts'

const PUBLIC_PAGE_PATHS: Record<PublicPageKey, string> = {
  home: '/',
  pricing: '/pricing',
  modelDetails: '/pricing/$modelId',
  rankings: '/rankings',
  about: '/about',
  privacyPolicy: '/privacy-policy',
  userAgreement: '/user-agreement',
}

const HOST_OWNED_PATHS = new Set([
  ...Object.values(PUBLIC_PAGE_PATHS),
  '/compare/ai-api-pricing',
  '/sign-in', '/sign-up', '/forgot-password', '/reset', '/user/reset', '/otp',
  '/setup', '/401', '/403', '/404', '/500', '/503',
])

const FORBIDDEN_PREFIXES = ['/oauth', '/console', '/dashboard', '/wallet', '/models',
  '/channels', '/keys', '/playground', '/profile', '/subscriptions', '/usage-logs',
  '/users', '/redemption-codes', '/system-settings', '/chat']

export function getPublicPagePath(page: PublicPageKey): string {
  return PUBLIC_PAGE_PATHS[page]
}

export function assertSkinRouteAllowed(path: string): void {
  if (!path.startsWith('/') || path.includes('//') || path.includes('?') || path.includes('#')) {
    throw new Error(`Invalid skin route path: ${path}`)
  }
  if (HOST_OWNED_PATHS.has(path) || FORBIDDEN_PREFIXES.some((prefix) => path === prefix || path.startsWith(`${prefix}/`))) {
    throw new Error(`Skin route conflicts with host-owned path: ${path}`)
  }
}
```

- [ ] **Step 5: Run the focused tests**

Run: `cd web/default && bun test src/skins/runtime/route-policy.test.ts`

Expected: 3 tests pass.

- [ ] **Step 6: Commit the policy contract**

```bash
git add web/default/src/skins/runtime
git commit -m "Keep skin routes outside host-owned application paths" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: bun test src/skins/runtime/route-policy.test.ts"
```

## Task 2: Implement Deterministic Skin Selection

**Files:**
- Create: `web/default/src/skins/default/build-manifest.ts`
- Create: `web/default/src/skins/custom/build-manifest.ts`
- Create: `web/default/scripts/skin-route-utils.ts`
- Create: `web/default/scripts/skin-route-utils.test.ts`
- Create: `web/default/scripts/generate-skin.ts`
- Create: `web/default/src/skins/runtime/active-skin.gen.ts`
- Create: `web/default/src/skins/runtime/active-skin-build.gen.ts`

- [ ] **Step 1: Write failing generator-selection tests**

```ts
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { normalizeSkinId, renderActiveSkinModule } from './skin-route-utils'

test('uses default when APP_SKIN is empty', () => {
  assert.equal(normalizeSkinId(undefined), 'default')
  assert.equal(normalizeSkinId(''), 'default')
})

test('rejects path traversal and unknown characters', () => {
  assert.throws(() => normalizeSkinId('../custom'))
  assert.throws(() => normalizeSkinId('custom skin'))
})

test('renders a static active manifest import', () => {
  assert.match(renderActiveSkinModule('custom'), /skins\/custom\/manifest/)
})
```

- [ ] **Step 2: Run the tests and verify failure**

Run: `cd web/default && bun test scripts/skin-route-utils.test.ts`

Expected: FAIL because the generator utilities do not exist.

- [ ] **Step 3: Add codegen-safe build manifests**

```ts
// src/skins/default/build-manifest.ts
import type { SkinBuildManifest } from '@/skins/runtime/build-contracts'

const defaultBuildManifest = {
  id: 'default',
  routes: [],
} as const satisfies SkinBuildManifest

export default defaultBuildManifest
```

```ts
// src/skins/custom/build-manifest.ts
import type { SkinBuildManifest } from '@/skins/runtime/build-contracts'

const customBuildManifest = {
  id: 'custom',
  routes: [],
} as const satisfies SkinBuildManifest

export default customBuildManifest
```

- [ ] **Step 4: Implement selection and generated-module rendering**

```ts
// scripts/skin-route-utils.ts
export function normalizeSkinId(value: string | undefined): string {
  const skinId = value?.trim() || 'default'
  if (!/^[a-z0-9-]+$/.test(skinId)) {
    throw new Error(`Invalid APP_SKIN value: ${skinId}`)
  }
  return skinId
}

export function renderActiveSkinModule(skinId: string): string {
  return `/* generated by scripts/generate-skin.ts */\nexport { default as activeSkin } from '@/skins/${skinId}/manifest'\n`
}

export function renderActiveBuildModule(skinId: string): string {
  return `/* generated by scripts/generate-skin.ts */\nexport { default as activeSkinBuild } from '@/skins/${skinId}/build-manifest'\n`
}
```

- [ ] **Step 5: Implement the generator entry point**

`generate-skin.ts` must:

1. normalize `process.env.APP_SKIN`;
2. verify `src/skins/<id>/manifest.ts` and `build-manifest.ts` exist;
3. write `active-skin.gen.ts` and `active-skin-build.gen.ts` only when content changes;
4. import the selected build manifest;
5. call the route-generation functions added in Task 3;
6. exit non-zero with a single actionable error message on invalid input.

Use `node:fs/promises`, `node:path`, and URL-safe paths so the same generator can
run from Bun's CLI and from Rsbuild's Node-based config evaluation. Do not use
Bun-only globals or shell string construction:

```ts
import { access } from 'node:fs/promises'

const root = new URL('../', import.meta.url)
const source = new URL(`src/skins/${skinId}/manifest.ts`, root)
try {
  await access(source)
} catch {
  throw new Error(`Unknown frontend skin: ${skinId}`)
}
```

- [ ] **Step 6: Commit default generated files**

The committed default files must contain static imports so a clean checkout can run TypeScript tooling before generation:

```ts
/* generated by scripts/generate-skin.ts */
export { default as activeSkin } from '@/skins/default/manifest'
```

```ts
/* generated by scripts/generate-skin.ts */
export { default as activeSkinBuild } from '@/skins/default/build-manifest'
```

- [ ] **Step 7: Run generator tests**

Run: `cd web/default && bun test scripts/skin-route-utils.test.ts`

Expected: 3 tests pass.

- [ ] **Step 8: Commit deterministic selection**

```bash
git add web/default/scripts web/default/src/skins
git commit -m "Make the selected frontend skin a build-time decision" \
  -m "Constraint: Each deployment produces one SPA bundle" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: bun test scripts/skin-route-utils.test.ts"
```

## Task 3: Generate Skin-Only TanStack Routes

**Files:**
- Modify: `web/default/scripts/skin-route-utils.ts`
- Modify: `web/default/scripts/skin-route-utils.test.ts`
- Modify: `web/default/scripts/generate-skin.ts`
- Create: `web/default/src/routes/(skin-generated)/.gitignore`

- [ ] **Step 1: Add failing path-to-file and collision tests**

```ts
import {
  getGeneratedRouteFile,
  renderGeneratedRoute,
  validateSkinRoutes,
} from './skin-route-utils'

test('maps static and dynamic URLs into a pathless route group', () => {
  assert.equal(getGeneratedRouteFile('/solutions'), '(skin-generated)/solutions.tsx')
  assert.equal(getGeneratedRouteFile('/guides/$slug'), '(skin-generated)/guides/$slug.tsx')
})

test('rejects duplicate and host-owned contributions', () => {
  assert.throws(() => validateSkinRoutes([
    { id: 'a', path: '/solutions', componentImport: './routes/a' },
    { id: 'b', path: '/solutions', componentImport: './routes/b' },
  ]))
  assert.throws(() => validateSkinRoutes([
    { id: 'pricing', path: '/pricing', componentImport: './routes/pricing' },
  ]))
})

test('renders a proxy through ActiveSkinRoute', () => {
  const source = renderGeneratedRoute({
    id: 'solutions',
    path: '/solutions',
    componentImport: './routes/solutions',
  })
  assert.match(source, /createFileRoute\('\/\(skin-generated\)\/solutions'\)/)
  assert.match(source, /<ActiveSkinRoute routeId='solutions' \/>/)
})
```

- [ ] **Step 2: Run the focused test and verify failure**

Run: `cd web/default && bun test scripts/skin-route-utils.test.ts`

Expected: FAIL because route generation functions are missing.

- [ ] **Step 3: Implement pure route generation functions**

Requirements:

- call `assertSkinRouteAllowed` for every path;
- reject duplicate IDs and duplicate paths;
- reject `.` and `..` path segments;
- map `/guides/$slug` to `(skin-generated)/guides/$slug.tsx`;
- render `createFileRoute('/(skin-generated)/guides/$slug')`;
- render only the route proxy; runtime component imports remain in the selected manifest;
- sort routes by path before writing for deterministic output.

The generated component body must be:

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { ActiveSkinRoute } from '@/skins/runtime/skin-route'

export const Route = createFileRoute('/(skin-generated)/solutions')({
  component: () => <ActiveSkinRoute routeId='solutions' />,
})
```

- [ ] **Step 4: Extend `generate-skin.ts` to replace generated proxies**

Use a dedicated generated directory. Delete only `.tsx` files beneath `src/routes/(skin-generated)`; never delete the directory or `.gitignore`. Create parent directories for nested paths, then write sorted proxy files.

- [ ] **Step 5: Add the generated-directory ignore rule**

```gitignore
*.tsx
!/.gitignore
```

- [ ] **Step 6: Run generator unit tests**

Run: `cd web/default && bun test scripts/skin-route-utils.test.ts`

Expected: all selection and route-generation tests pass.

- [ ] **Step 7: Smoke-test both empty manifests**

Run:

```bash
cd web/default
APP_SKIN=default bun scripts/generate-skin.ts
APP_SKIN=custom bun scripts/generate-skin.ts
find 'src/routes/(skin-generated)' -type f -maxdepth 3 -print
```

Expected: only `.gitignore` exists because both initial build manifests have no route contributions.

- [ ] **Step 8: Commit route generation**

```bash
git add web/default/scripts web/default/src/routes/'(skin-generated)' web/default/src/skins/runtime
git commit -m "Keep skin-only routes type-safe at build time" \
  -m "Rejected: Runtime catch-all routing | bypasses TanStack route typing and 404 behavior" \
  -m "Confidence: high" \
  -m "Scope-risk: moderate" \
  -m "Tested: generator unit tests and default/custom generation smoke tests"
```

## Task 4: Add Default and Custom Runtime Manifests

**Files:**
- Create: `web/default/src/skins/default/manifest.ts`
- Create: `web/default/src/skins/custom/manifest.ts`
- Create: `web/default/src/skins/custom/shell/custom-public-shell.tsx`
- Create: `web/default/src/skins/custom/styles/index.css`
- Create: `web/default/src/skins/runtime/resolve-page.ts`
- Create: `web/default/src/skins/runtime/resolve-page.test.ts`

- [ ] **Step 1: Write failing page-resolution tests**

```ts
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { resolveSkinPage } from './resolve-page'
import type { ThemeManifest } from './contracts'

const component = () => null
const defaultManifest = {
  id: 'default',
  build: { id: 'default', routes: [] },
  pages: { home: { component, shell: 'self' } },
  routes: [],
} satisfies ThemeManifest

test('uses the active skin override when present', () => {
  const active = {
    ...defaultManifest,
    id: 'custom',
    build: { id: 'custom', routes: [] },
    pages: { home: { component, shell: 'skin' as const } },
  }
  assert.equal(resolveSkinPage(active, defaultManifest, 'home').shell, 'skin')
})

test('falls back to the default page when the active skin omits it', () => {
  const active = { ...defaultManifest, id: 'custom', pages: {} }
  assert.equal(resolveSkinPage(active, defaultManifest, 'home').shell, 'self')
})
```

- [ ] **Step 2: Run the test and verify failure**

Run: `cd web/default && bun test src/skins/runtime/resolve-page.test.ts`

Expected: FAIL because `resolveSkinPage` does not exist.

- [ ] **Step 3: Implement the default manifest as adapters**

Map the current feature exports without changing them:

```ts
const manifest: ThemeManifest = {
  id: 'default',
  build: defaultBuildManifest,
  pages: {
    home: { component: Home, shell: 'self' },
    pricing: { component: Pricing, shell: 'self' },
    modelDetails: { component: ModelDetails, shell: 'self' },
    rankings: { component: Rankings, shell: 'self' },
    about: { component: About, shell: 'self' },
    privacyPolicy: { component: PrivacyPolicy, shell: 'self' },
    userAgreement: { component: UserAgreement, shell: 'self' },
  },
  routes: [],
}

export default manifest
```

- [ ] **Step 4: Implement the custom scaffold without visual overrides**

```ts
const manifest: ThemeManifest = {
  id: 'custom',
  build: customBuildManifest,
  pages: {},
  routes: [],
  shell: CustomPublicShell,
}

export default manifest
```

`CustomPublicShell` must import `../styles/index.css` and render only
`props.children`. `SkinBoundary`, not the shell, owns the portal root. Do not
redesign Header or Footer in this foundation plan.

- [ ] **Step 5: Implement strict page resolution**

```ts
export function resolveSkinPage(
  activeSkin: ThemeManifest,
  defaultSkin: ThemeManifest,
  page: PublicPageKey
): SkinPageDefinition {
  const definition = activeSkin.pages[page] ?? defaultSkin.pages[page]
  if (!definition) throw new Error(`No page registered for ${page}`)
  return definition
}
```

- [ ] **Step 6: Run page-resolution tests**

Run: `cd web/default && bun test src/skins/runtime/resolve-page.test.ts`

Expected: 2 tests pass.

- [ ] **Step 7: Commit runtime manifests**

```bash
git add web/default/src/skins
git commit -m "Preserve current public pages behind a skin manifest" \
  -m "Constraint: The custom scaffold must not change production visuals" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: bun test src/skins/runtime/resolve-page.test.ts"
```

## Task 5: Implement SkinBoundary and Portal Ownership

**Files:**
- Create: `web/default/src/skins/runtime/skin-portal.tsx`
- Create: `web/default/src/skins/runtime/skin-boundary.tsx`
- Create: `web/default/src/skins/runtime/skin-boundary.test.tsx`

- [ ] **Step 1: Write a server-rendered boundary test**

```tsx
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { SkinBoundary } from './skin-boundary'

test('scopes skin content and creates a portal root', () => {
  const html = renderToStaticMarkup(
    <SkinBoundary skinId='custom'><span>content</span></SkinBoundary>
  )
  assert.match(html, /data-skin="custom"/)
  assert.match(html, /data-skin-portal-root="custom"/)
  assert.match(html, />content</)
})
```

- [ ] **Step 2: Run the test and verify failure**

Run: `cd web/default && bun test src/skins/runtime/skin-boundary.test.tsx`

Expected: FAIL because `SkinBoundary` does not exist.

- [ ] **Step 3: Implement portal context and scoped wrapper**

`SkinBoundary` must render one root element with `data-skin`, render one descendant portal container with `data-skin-portal-root`, and expose that element through `useSkinPortalContainer()` after mount. It may set `document.body.dataset.skinSurface` while mounted, but must remove it in effect cleanup only when the current value still matches its own skin ID.

Do not override `:root`, `body`, or `.dark` in skin CSS. The custom stylesheet must start with:

```css
@layer theme {
  [data-skin='custom'] {
    --skin-font-body: var(--font-sans);
  }
}
```

- [ ] **Step 4: Run the boundary test**

Run: `cd web/default && bun test src/skins/runtime/skin-boundary.test.tsx`

Expected: 1 test passes.

- [ ] **Step 5: Commit the style boundary**

```bash
git add web/default/src/skins/runtime web/default/src/skins/custom/styles
git commit -m "Contain product-skin styles within public route boundaries" \
  -m "Directive: Skin overlays must use the skin portal container" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: server-rendered SkinBoundary test"
```

## Task 6: Delegate Existing Public Routes Through SkinPage

**Files:**
- Create: `web/default/src/skins/runtime/skin-page.tsx`
- Modify: `web/default/src/routes/index.tsx`
- Modify: `web/default/src/routes/pricing/index.tsx`
- Modify: `web/default/src/routes/pricing/$modelId/index.tsx`
- Modify: `web/default/src/routes/compare/ai-api-pricing/index.tsx`
- Modify: `web/default/src/routes/rankings/index.tsx`
- Modify: `web/default/src/routes/about/index.tsx`
- Modify: `web/default/src/routes/privacy-policy.tsx`
- Modify: `web/default/src/routes/user-agreement.tsx`

- [ ] **Step 1: Implement `SkinPage` with explicit fallback behavior**

```tsx
import { Suspense, createElement } from 'react'
import defaultSkin from '@/skins/default/manifest'
import { activeSkin } from './active-skin.gen'
import type { PublicPageKey } from './contracts'
import { resolveSkinPage } from './resolve-page'
import { SkinBoundary } from './skin-boundary'

export function SkinPage(props: { page: PublicPageKey }) {
  const definition = resolveSkinPage(activeSkin, defaultSkin, props.page)
  const content = (
    <Suspense fallback={null}>{createElement(definition.component)}</Suspense>
  )
  if (definition.shell === 'self' || !activeSkin.shell) return content
  return (
    <SkinBoundary skinId={activeSkin.id}>
      {createElement(activeSkin.shell, { children: content })}
    </SkinBoundary>
  )
}
```

- [ ] **Step 2: Replace direct feature imports in each approved route**

Example for the homepage:

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { SkinPage } from '@/skins/runtime/skin-page'

export const Route = createFileRoute('/')({
  component: () => <SkinPage page='home' />,
})
```

For pricing, model detail, and rankings, preserve the existing Zod search schemas unchanged and replace only the `component` field. `/compare/ai-api-pricing` delegates to `page='pricing'`.

- [ ] **Step 3: Prove auth and system routes were not modified**

Run:

```bash
git diff --name-only -- web/default/src/routes | rg '\(auth\)|\(errors\)|setup|_authenticated'
```

Expected: no output.

- [ ] **Step 4: Generate the default selection and typecheck**

Run:

```bash
cd web/default
APP_SKIN=default bun scripts/generate-skin.ts
bun run typecheck
```

Expected: exit 0 with no TypeScript errors.

- [ ] **Step 5: Generate the custom scaffold and typecheck**

Run:

```bash
cd web/default
APP_SKIN=custom bun scripts/generate-skin.ts
bun run typecheck
```

Expected: exit 0; all public pages resolve through default fallback.

- [ ] **Step 6: Restore committed generated default selection**

Run: `cd web/default && APP_SKIN=default bun scripts/generate-skin.ts`

Expected: `active-skin.gen.ts` points to default.

- [ ] **Step 7: Commit route delegation**

```bash
git add web/default/src/routes web/default/src/skins/runtime
git commit -m "Let approved public routes resolve through the selected skin" \
  -m "Constraint: Route schemas and protected route behavior remain host-owned" \
  -m "Confidence: high" \
  -m "Scope-risk: moderate" \
  -m "Tested: default and custom TypeScript checks"
```

## Task 7: Resolve Generated Skin Routes at Runtime

**Files:**
- Create: `web/default/src/skins/runtime/skin-route.tsx`
- Create: `web/default/src/skins/runtime/skin-route.test.ts`

- [ ] **Step 1: Write failing route-resolution tests**

```ts
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { resolveRuntimeRoute } from './skin-route'

test('resolves an active skin route by ID', () => {
  const component = () => null
  assert.equal(resolveRuntimeRoute([{ id: 'solutions', component }], 'solutions'), component)
})

test('throws for generated routes missing from the runtime manifest', () => {
  assert.throws(() => resolveRuntimeRoute([], 'missing'), /missing/)
})
```

- [ ] **Step 2: Run the tests and verify failure**

Run: `cd web/default && bun test src/skins/runtime/skin-route.test.ts`

Expected: FAIL because route resolution does not exist.

- [ ] **Step 3: Implement strict route resolution and rendering**

```tsx
export function resolveRuntimeRoute(routes: readonly SkinRuntimeRoute[], routeId: string) {
  const route = routes.find((item) => item.id === routeId)
  if (!route) throw new Error(`Active skin route is not registered: ${routeId}`)
  return route.component
}

export function ActiveSkinRoute(props: { routeId: string }) {
  const Component = resolveRuntimeRoute(activeSkin.routes, props.routeId)
  const content = <Suspense fallback={null}><Component /></Suspense>
  if (!activeSkin.shell) return content
  const Shell = activeSkin.shell
  return (
    <SkinBoundary skinId={activeSkin.id}>
      <Shell>{content}</Shell>
    </SkinBoundary>
  )
}
```

- [ ] **Step 4: Run the focused tests**

Run: `cd web/default && bun test src/skins/runtime/skin-route.test.ts`

Expected: 2 tests pass.

- [ ] **Step 5: Add a test-only route contribution fixture**

In `skin-route-utils.test.ts`, generate `/guides/$slug` into a temporary directory, assert the file path and source, then remove the temporary directory in test cleanup. Do not add a production custom route merely to exercise the generator.

- [ ] **Step 6: Commit runtime route resolution**

```bash
git add web/default/src/skins/runtime web/default/scripts/skin-route-utils.test.ts
git commit -m "Resolve generated routes only from the active skin manifest" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: runtime route and temporary generation tests"
```

## Task 8: Merge Skin Navigation Without Bypassing Host Rules

**Files:**
- Modify: `web/default/src/skins/runtime/contracts.ts`
- Create: `web/default/src/skins/runtime/navigation.ts`
- Create: `web/default/src/skins/runtime/navigation.test.ts`
- Modify: `web/default/src/hooks/use-top-nav-links.ts`
- Modify: `web/default/src/hooks/use-top-nav-links.test.ts`

- [ ] **Step 1: Write failing merge tests**

```ts
test('places skin header links after host links in declared order', () => {
  const links = mergeSkinHeaderLinks(
    [{ title: 'Home', href: '/' }],
    [
      { labelKey: 'Solutions', href: '/solutions', order: 20 },
      { labelKey: 'Compare', href: '/compare', order: 10 },
    ],
    identity
  )
  assert.deepEqual(links.map((link) => link.title), ['Home', 'Compare', 'Solutions'])
})

test('does not add a skin link whose route is absent', () => {
  assert.throws(() => mergeSkinHeaderLinks([], [
    { labelKey: 'Missing', href: '/missing', order: 1 },
  ], identity, new Set(['/solutions'])))
})
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd web/default && bun test src/skins/runtime/navigation.test.ts`

Expected: FAIL because navigation merging does not exist.

- [ ] **Step 3: Implement navigation contributions**

Add `navigation.header` and `navigation.footer` arrays to `ThemeManifest`. `mergeSkinHeaderLinks` must translate `labelKey`, preserve host-generated `disabled`, `requireAuth`, module toggles, and custom-link behavior, and append only validated active-skin links sorted by `order` then `href`.

- [ ] **Step 4: Integrate after existing host link construction**

Do not replace `buildTopNavLinks`. Call the merge helper after it has applied backend module settings. This ensures pricing/rankings `requireAuth` and administrator visibility controls remain authoritative.

- [ ] **Step 5: Run navigation regression tests**

Run:

```bash
cd web/default
bun test src/skins/runtime/navigation.test.ts src/hooks/use-top-nav-links.test.ts
```

Expected: all new tests and existing top-nav tests pass.

- [ ] **Step 6: Commit navigation integration**

```bash
git add web/default/src/skins/runtime web/default/src/hooks/use-top-nav-links.ts web/default/src/hooks/use-top-nav-links.test.ts
git commit -m "Allow skins to extend public navigation without bypassing host policy" \
  -m "Directive: Backend module and requireAuth settings remain authoritative" \
  -m "Confidence: high" \
  -m "Scope-risk: narrow" \
  -m "Tested: skin navigation and existing top-nav regression tests"
```

## Task 9: Wire Generation Into Dev, Typecheck, and Production Builds

**Files:**
- Modify: `web/default/package.json`
- Modify: `web/default/rsbuild.config.ts`
- Modify: `web/default/knip.config.ts`

- [ ] **Step 1: Add explicit scripts**

```json
{
  "scripts": {
    "skin:generate": "bun scripts/generate-skin.ts",
    "dev": "rsbuild dev",
    "build": "rsbuild build",
    "build:check": "bun run typecheck && rsbuild build",
    "typecheck": "bun run skin:generate && tsc -b",
    "build:default": "APP_SKIN=default bun run build:check",
    "build:custom": "APP_SKIN=custom bun run build:check"
  }
}
```

Keep every unrelated existing script unchanged.

- [ ] **Step 2: Make Rsbuild generation ordering explicit**

Because the TanStack plugin discovers route files during config evaluation,
call the generator before returning the Rsbuild config. Extract a callable
`generateSkin()` from `scripts/generate-skin.ts` and invoke it at the top of the
async `defineConfig` callback before the `tanstackRouter` plugin is constructed.
This is the only generation entry for `dev` and `build`; the explicit
`skin:generate` script exists for typecheck and diagnostics.

The CLI entry point must still work:

```ts
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const isDirectRun = process.argv[1]
  ? path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
  : false

if (isDirectRun) await generateSkin()
```

- [ ] **Step 3: Ignore generated sources in Knip**

Add:

```ts
'src/skins/runtime/*.gen.ts',
'src/routes/(skin-generated)/**',
```

Do not ignore the non-generated skin runtime or manifests.

- [ ] **Step 4: Verify unknown skins fail clearly**

Run: `cd web/default && APP_SKIN=missing bun run skin:generate`

Expected: non-zero exit and `Unknown frontend skin: missing`.

- [ ] **Step 5: Run both full build checks**

Run:

```bash
cd web/default
bun run build:default
bun run build:custom
```

Expected: both commands exit 0.

- [ ] **Step 6: Restore default generated selection**

Run: `cd web/default && APP_SKIN=default bun run skin:generate`

- [ ] **Step 7: Commit build integration**

```bash
git add web/default/package.json web/default/rsbuild.config.ts web/default/knip.config.ts web/default/scripts/generate-skin.ts web/default/src/skins/runtime/*.gen.ts
git commit -m "Generate the selected skin before every frontend build" \
  -m "Constraint: TanStack must see generated route files during config evaluation" \
  -m "Confidence: high" \
  -m "Scope-risk: moderate" \
  -m "Tested: default and custom build checks plus unknown-skin failure"
```

## Task 10: Final Regression and Browser Verification

**Files:**
- Modify only if verification exposes a defect in files already owned by Tasks 1-9.

- [ ] **Step 1: Run all focused skin tests**

Run:

```bash
cd web/default
bun test scripts/skin-route-utils.test.ts \
  src/skins/runtime/route-policy.test.ts \
  src/skins/runtime/resolve-page.test.ts \
  src/skins/runtime/skin-boundary.test.tsx \
  src/skins/runtime/skin-route.test.ts \
  src/skins/runtime/navigation.test.ts \
  src/hooks/use-top-nav-links.test.ts
```

Expected: zero failures.

- [ ] **Step 2: Run repository frontend checks**

Run:

```bash
cd web/default
APP_SKIN=default bun run typecheck
APP_SKIN=default bun run lint
APP_SKIN=default bun run format:check
APP_SKIN=default bun run build
APP_SKIN=custom bun run typecheck
APP_SKIN=custom bun run build
```

Expected: every command exits 0. If repository-wide lint or format reports pre-existing failures, record exact files and run scoped checks for every changed file; do not claim the broad check passed.

- [ ] **Step 3: Start the default development build**

Run: `cd web/default && APP_SKIN=default bun run dev`

Expected: Rsbuild prints a reachable local URL.

- [ ] **Step 4: Verify default routes in a browser**

Check desktop and mobile widths for:

- `/`;
- `/pricing`;
- one `/pricing/$modelId` link reached from the marketplace;
- `/rankings`;
- `/about`;
- `/privacy-policy`;
- `/user-agreement`;
- `/sign-in`;
- `/404`.

Expected: approved public routes retain current appearance and behavior; auth/error routes are unchanged; browser console has no route or lazy-import errors.

- [ ] **Step 5: Start and verify the custom scaffold**

Run: `cd web/default && APP_SKIN=custom bun run dev`

Repeat the same route checks. Expected: every existing public page falls back to the default implementation, and `/sign-in` plus `/404` remain outside `data-skin='custom'`.

- [ ] **Step 6: Verify route cleanup and repository diff**

Run:

```bash
cd web/default
APP_SKIN=default bun run skin:generate
git diff --check
git status --short
```

Expected: no generated custom route proxy remains, no whitespace errors, and only intentional implementation files are changed.

- [ ] **Step 7: Commit any verification-only corrections**

If corrections were required, commit only those corrections with a Lore message that names the failed verification and the exact checks rerun. If no corrections were required, do not create an empty commit.

## Follow-up Plans After Foundation

Once this plan is complete and verified, create separate design/implementation plans in this order:

1. `custom-public-shell-and-home`: visual direction, Header, Footer, homepage, assets, responsive and accessibility checks;
2. `public-model-catalog-domain`: extract pricing/model data contracts and preserve current URL/search behavior with regression tests;
3. `custom-model-marketplace`: custom listing, filters, card/table views, loading/empty/error states, and model detail;
4. `custom-public-secondary-pages`: rankings, about, legal pages, `/compare/ai-api-pricing`, and the first real skin-only route.

Do not begin those plans until their visual and product requirements are approved. The foundation is successful when a third skin can be added through manifests and skin-owned files without another host-router redesign.
