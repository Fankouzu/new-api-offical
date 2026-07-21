import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type {
  DefaultThemeManifest,
  PublicPageKey,
  SkinPageDefinition,
  ThemeManifest,
} from './contracts'
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

function createDefaultManifest(
  pages: Partial<Record<PublicPageKey, SkinPageDefinition>> = {}
): DefaultThemeManifest {
  const fallbackPage: SkinPageDefinition = {
    component: DefaultHome,
    shell: 'self',
  }

  return {
    id: 'default',
    build: { id: 'default', routes: [] },
    pages: {
      home: fallbackPage,
      pricing: fallbackPage,
      modelDetails: fallbackPage,
      rankings: fallbackPage,
      about: fallbackPage,
      privacyPolicy: fallbackPage,
      userAgreement: fallbackPage,
      ...pages,
    },
    routes: [],
  }
}

describe('resolveSkinPage', () => {
  it('prefers the active skin page override', () => {
    const activeSkin = createManifest('custom', {
      home: { component: ActiveHome, shell: 'skin' },
    })
    const defaultSkin = createDefaultManifest({
      home: { component: DefaultHome, shell: 'self' },
    })

    const resolvedPage = resolveSkinPage(activeSkin, defaultSkin, 'home')

    assert.equal(resolvedPage.component, ActiveHome)
    assert.equal(resolvedPage.shell, 'skin')
  })

  it('falls back to the default skin page', () => {
    const activeSkin = createManifest('custom', {})
    const defaultSkin = createDefaultManifest({
      pricing: { component: DefaultPricing, shell: 'self' },
    })

    const resolvedPage = resolveSkinPage(activeSkin, defaultSkin, 'pricing')

    assert.equal(resolvedPage.component, DefaultPricing)
    assert.equal(resolvedPage.shell, 'self')
  })

  it('throws an actionable error when neither manifest defines the page', () => {
    const page: PublicPageKey = 'about'
    const defaultSkin = createDefaultManifest()
    const pagesWithoutAbout = new Proxy(defaultSkin.pages, {
      get(target, property, receiver) {
        if (property === page) {
          return undefined
        }

        return Reflect.get(target, property, receiver)
      },
    })

    assert.throws(
      () =>
        resolveSkinPage(
          createManifest('custom', {}),
          { ...defaultSkin, pages: pagesWithoutAbout },
          page
        ),
      {
        message:
          'No page registered for about in active skin custom or default skin default',
      }
    )
  })

  it('does not mutate either manifest', () => {
    const activePages = Object.freeze({})
    const defaultSkin = createDefaultManifest({
      home: Object.freeze({ component: DefaultHome, shell: 'self' }),
    })
    const defaultPages = Object.freeze(defaultSkin.pages)
    const activeSkin = Object.freeze(createManifest('custom', activePages))
    const frozenDefaultSkin = Object.freeze({
      ...defaultSkin,
      pages: defaultPages,
    })

    const resolvedPage = resolveSkinPage(activeSkin, frozenDefaultSkin, 'home')

    assert.equal(resolvedPage, defaultPages.home)
    assert.deepEqual(activeSkin.pages, {})
    assert.deepEqual(defaultSkin.pages, defaultPages)
  })
})
