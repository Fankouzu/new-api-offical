import { createElement } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { SkinBoundary } from './skin-boundary'
import { SkinPortalProvider } from './skin-portal'
import { useSkinPortalContainer } from './use-skin-portal-container'

describe('SkinBoundary', () => {
  it('renders the skin boundary, portal root, and children on the server', () => {
    const markup = renderToStaticMarkup(
      createElement(SkinBoundary, {
        skinId: 'custom',
        children: createElement('main', null, 'Custom content'),
      })
    )

    assert.match(markup, /^<div[^>]*data-skin="custom"[^>]*>/)
    assert.match(markup, /^<div[^>]*data-skin-boundary="true"[^>]*>/)
    assert.match(markup, /<main>Custom content<\/main>/)
    assert.match(markup, /<div data-skin-portal-root="custom"><\/div>/)
    assert.equal((markup.match(/data-skin-boundary=/g) ?? []).length, 1)
    assert.equal((markup.match(/data-skin-portal-root=/g) ?? []).length, 1)
    assert.ok(
      markup.indexOf('data-skin-portal-root') >
        markup.indexOf('data-skin-boundary')
    )
  })

  it('rejects portal hook use outside a SkinBoundary provider', () => {
    function PortalConsumer() {
      useSkinPortalContainer()
      return createElement('span', null, 'unreachable')
    }

    assert.throws(
      () => renderToStaticMarkup(createElement(PortalConsumer)),
      /useSkinPortalContainer must be used within a SkinBoundary/
    )
  })

  it('exposes a nullable portal container inside the provider', () => {
    function PortalConsumer() {
      return createElement(
        'span',
        null,
        useSkinPortalContainer() === null ? 'pending' : 'ready'
      )
    }

    const markup = renderToStaticMarkup(
      createElement(SkinPortalProvider, {
        container: null,
        children: createElement(PortalConsumer),
      })
    )

    assert.equal(markup, '<span>pending</span>')
  })
})
