import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { PublicPageKey, ThemeManifest } from './contracts'
import { resolveSkinPage } from './resolve-page'

const ActiveHome: ComponentType = () => null
const DefaultHome: ComponentType = () => null
const DefaultPricing: ComponentType = () => null

function createManifest(
  id: string,
  pages: ThemeManifest['pages']
): ThemeManifest {
  return {
    id,
    build: { id, routes: [] },
    pages,
    routes: [],
  }
}

describe('resolveSkinPage', () => {
  it('prefers the active skin page override', () => {
    const activeSkin = createManifest('custom', {
      home: { component: ActiveHome, shell: 'skin' },
    })
    const defaultSkin = createManifest('default', {
      home: { component: DefaultHome, shell: 'self' },
    })

    const resolvedPage = resolveSkinPage(activeSkin, defaultSkin, 'home')

    assert.equal(resolvedPage.component, ActiveHome)
    assert.equal(resolvedPage.shell, 'skin')
  })

  it('falls back to the default skin page', () => {
    const activeSkin = createManifest('custom', {})
    const defaultSkin = createManifest('default', {
      pricing: { component: DefaultPricing, shell: 'self' },
    })

    const resolvedPage = resolveSkinPage(activeSkin, defaultSkin, 'pricing')

    assert.equal(resolvedPage.component, DefaultPricing)
    assert.equal(resolvedPage.shell, 'self')
  })

  it('throws an actionable error when neither manifest defines the page', () => {
    const page: PublicPageKey = 'about'

    assert.throws(
      () =>
        resolveSkinPage(
          createManifest('custom', {}),
          createManifest('default', {}),
          page
        ),
      { message: `No page registered for ${page}` }
    )
  })

  it('does not mutate either manifest', () => {
    const activePages = Object.freeze({})
    const defaultPages = Object.freeze({
      home: Object.freeze({ component: DefaultHome, shell: 'self' as const }),
    })
    const activeSkin = Object.freeze(createManifest('custom', activePages))
    const defaultSkin = Object.freeze(createManifest('default', defaultPages))

    const resolvedPage = resolveSkinPage(activeSkin, defaultSkin, 'home')

    assert.equal(resolvedPage, defaultPages.home)
    assert.deepEqual(activeSkin.pages, {})
    assert.deepEqual(defaultSkin.pages, defaultPages)
  })
})
