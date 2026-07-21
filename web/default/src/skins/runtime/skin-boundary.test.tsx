import { createElement } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import {
  clearOwnedSkinSurface,
  setSkinSurface,
  SkinBoundary,
} from './skin-boundary'
import { useSkinPortalContainer } from './skin-portal'

describe('SkinBoundary', () => {
  it('renders the skin boundary, portal root, and children on the server', () => {
    const markup = renderToStaticMarkup(
      createElement(SkinBoundary, {
        skinId: 'custom',
        children: createElement('main', null, 'Custom content'),
      })
    )

    assert.match(markup, /^<div data-skin="custom" data-skin-boundary="true">/)
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
})

describe('skin surface ownership', () => {
  it('sets the body marker', () => {
    const dataset: { skinSurface?: string } = {}

    setSkinSurface(dataset, 'custom')

    assert.equal(dataset.skinSurface, 'custom')
  })

  it('does not clear a marker owned by a newer boundary', () => {
    const dataset: { skinSurface?: string } = { skinSurface: 'newer' }

    clearOwnedSkinSurface(dataset, 'custom')

    assert.equal(dataset.skinSurface, 'newer')
  })

  it('clears its matching marker', () => {
    const dataset: { skinSurface?: string } = { skinSurface: 'custom' }

    clearOwnedSkinSurface(dataset, 'custom')

    assert.equal(dataset.skinSurface, undefined)
  })
})
