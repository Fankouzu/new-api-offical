import React, { createElement } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { ActiveSkinRoute } from './skin-route'

globalThis.React = React

describe('ActiveSkinRoute', () => {
  it('uses the committed active skin manifest when resolving a generated route', () => {
    assert.throws(
      () =>
        renderToStaticMarkup(
          createElement(ActiveSkinRoute, {
            routeId: 'missing',
            routePath: '/missing',
          })
        ),
      {
        message:
          'Skin default route missing at generated path /missing is missing from the build manifest',
      }
    )
  })
})
