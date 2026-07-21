# Custom Home Hero Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Display a non-interactive red dashed boundary over the first 850px of the custom-skin home document, including navigation.

**Architecture:** Override only the custom manifest's `home` page with a thin wrapper around the existing shared `Home` component. The wrapper supplies a home-only DOM marker; custom skin CSS uses that marker to draw an absolute document-level measurement overlay without changing layout or pointer behavior.

**Tech Stack:** React 19, TypeScript, CSS, Node test runner, React DOM server rendering

---

### Task 1: Add the custom home measurement marker

**Files:**
- Create: `web/default/src/skins/custom/pages/custom-home.tsx`
- Create: `web/default/src/skins/custom/pages/custom-home.test.tsx`
- Modify: `web/default/src/skins/custom/manifest.ts`
- Modify: `web/default/src/skins/custom/styles/index.css`

- [ ] **Step 1: Write the failing wrapper test**

Create a server-rendering test that stubs CSS loading, renders `CustomHome`, and asserts that the output has `data-skin-page="home"` around the shared home-page boundary.

- [ ] **Step 2: Run the focused test and verify failure**

Run:

```bash
bunx tsx --test src/skins/custom/pages/custom-home.test.tsx
```

Expected: failure because `custom-home.tsx` does not exist.

- [ ] **Step 3: Implement the wrapper and manifest override**

Create a component equivalent to:

```tsx
import { Home } from '@/features/home'

export function CustomHome() {
  return (
    <div data-skin-page='home'>
      <Home />
    </div>
  )
}
```

Register it as the custom manifest's `home` page with `shell: 'self'`, keeping every other page on default fallback.

- [ ] **Step 4: Add the temporary boundary CSS**

Add a custom-only, home-only document pseudo-element with:

```css
body[data-skin='custom']:has([data-skin-page='home'])::before {
  position: absolute;
  z-index: 9999;
  inset: 0 0 auto;
  height: 850px;
  border: 4px dashed #ef4444;
  pointer-events: none;
  content: '';
}
```

The overlay starts at document y=0, includes navigation, does not affect layout, and scrolls with the document.

- [ ] **Step 5: Verify focused and manifest behavior**

Run:

```bash
bunx tsx --test src/skins/custom/pages/custom-home.test.tsx src/skins/custom/shell/custom-public-shell.test.tsx
bun run typecheck
```

Expected: all focused tests and typecheck pass.

- [ ] **Step 6: Verify the running custom page**

Open `http://localhost:3107/` and confirm the red dashed boundary starts at the document top, ends at 850px, includes navigation, and does not appear on `/pricing`.

- [ ] **Step 7: Commit**

```bash
git add web/default/src/skins/custom/pages/custom-home.tsx \
  web/default/src/skins/custom/pages/custom-home.test.tsx \
  web/default/src/skins/custom/manifest.ts \
  web/default/src/skins/custom/styles/index.css
git commit -m "Expose the approved custom hero review boundary"
```
