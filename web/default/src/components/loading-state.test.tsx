import React, { createElement } from 'react'
import i18next from 'i18next'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { initReactI18next } from 'react-i18next'
import { LoadingState } from './loading-state'

globalThis.React = React
await i18next
  .use(initReactI18next)
  .init({ lng: 'en', resources: {}, showSupportNotice: false })

describe('LoadingState', () => {
  it('announces non-inline loading progress accessibly', () => {
    const markup = renderToStaticMarkup(
      createElement(LoadingState, { message: 'Loading pricing' })
    )

    assert.match(markup, /role="status"/)
    assert.match(markup, /aria-live="polite"/)
    assert.match(markup, /aria-busy="true"/)
    assert.match(markup, /aria-hidden="true"/)
    assert.match(markup, />Loading pricing</)
  })
})
