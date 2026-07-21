import React, { createElement, lazy } from 'react'
import i18next from 'i18next'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup, renderToString } from 'react-dom/server'
import { initReactI18next } from 'react-i18next'
import type { SkinPageRenderPlan } from './skin-page-plan'
import { SkinPageRenderer } from './skin-page-renderer'

globalThis.React = React
await i18next
  .use(initReactI18next)
  .init({ lng: 'en', resources: {}, showSupportNotice: false })

function Page() {
  return createElement('main', null, 'Page content')
}

function Shell(props: { children: React.ReactNode }) {
  return createElement('section', { 'data-test-shell': true }, props.children)
}

describe('SkinPageRenderer', () => {
  it('renders self-owned pages without a skin boundary or shell', () => {
    const plan: SkinPageRenderPlan = {
      shell: 'self',
      definition: { component: Page, shell: 'self' },
    }

    const markup = renderToStaticMarkup(
      createElement(SkinPageRenderer, { plan })
    )

    assert.match(markup, /<main>Page content<\/main>/)
    assert.doesNotMatch(markup, /data-skin-boundary/)
    assert.doesNotMatch(markup, /data-test-shell/)
  })

  it('renders skin-owned pages inside one boundary and the selected shell', () => {
    const plan: SkinPageRenderPlan = {
      shell: 'skin',
      definition: { component: Page, shell: 'skin' },
      skinId: 'custom',
      Shell,
    }

    const markup = renderToStaticMarkup(
      createElement(SkinPageRenderer, { plan })
    )

    assert.match(markup, /data-skin-boundary="true"/)
    assert.match(markup, /data-test-shell="true"/)
    assert.equal((markup.match(/data-skin-boundary=/g) ?? []).length, 1)
  })

  it('renders the accessible loading fallback while a lazy page is pending', () => {
    const PendingPage = lazy(() => new Promise<never>(() => undefined))
    const plan: SkinPageRenderPlan = {
      shell: 'self',
      definition: { component: PendingPage, shell: 'self' },
    }

    const markup = renderToString(createElement(SkinPageRenderer, { plan }))

    assert.match(markup, /role="status"/)
    assert.match(markup, /aria-live="polite"/)
    assert.match(markup, />Loading\.\.\.</)
  })
})
