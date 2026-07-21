import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type {
  DefaultThemeManifest,
  PublicPageKey,
  SkinPageDefinition,
  ThemeManifest,
} from './contracts'
import { createSkinPageRenderPlan } from './skin-page-plan'

const Page: ComponentType = () => null
const Shell: ComponentType<{ children: React.ReactNode }> = () => null

function createManifest(
  id: string,
  pages: ThemeManifest['pages'],
  shell?: ThemeManifest['shell']
): ThemeManifest {
  return {
    id,
    build: { id, routes: [] },
    pages,
    routes: [],
    shell,
  }
}

function createDefaultManifest(
  page: PublicPageKey,
  definition: SkinPageDefinition
): DefaultThemeManifest {
  const fallback = { component: Page, shell: 'self' } as const

  return {
    id: 'default',
    build: { id: 'default', routes: [] },
    pages: {
      home: fallback,
      pricing: fallback,
      modelDetails: fallback,
      rankings: fallback,
      about: fallback,
      privacyPolicy: fallback,
      userAgreement: fallback,
      [page]: definition,
    },
    routes: [],
  }
}

describe('createSkinPageRenderPlan', () => {
  it('keeps a default fallback self-owned without a boundary or skin shell', () => {
    const defaultPage = { component: Page, shell: 'self' } as const

    const plan = createSkinPageRenderPlan(
      createManifest('custom', {}),
      createDefaultManifest('pricing', defaultPage),
      'pricing'
    )

    assert.equal(plan.definition, defaultPage)
    assert.equal(plan.shell, 'self')
    assert.equal('skinId' in plan, false)
    assert.equal('Shell' in plan, false)
  })

  it('selects the active skin shell for a skin-owned page', () => {
    const activePage = { component: Page, shell: 'skin' } as const

    const plan = createSkinPageRenderPlan(
      createManifest('custom', { home: activePage }, Shell),
      createDefaultManifest('home', { component: Page, shell: 'self' }),
      'home'
    )

    assert.equal(plan.definition, activePage)
    assert.equal(plan.shell, 'skin')
    if (plan.shell !== 'skin') {
      assert.fail('Expected a skin-owned render plan')
    }
    assert.equal(plan.skinId, 'custom')
    assert.equal(plan.Shell, Shell)
  })

  it('rejects a skin-owned page when its manifest has no shell', () => {
    assert.throws(
      () =>
        createSkinPageRenderPlan(
          createManifest('custom', {
            about: { component: Page, shell: 'skin' },
          }),
          createDefaultManifest('about', {
            component: Page,
            shell: 'self',
          }),
          'about'
        ),
      {
        message:
          'Skin custom page about requires a skin shell, but none is configured',
      }
    )
  })
})
