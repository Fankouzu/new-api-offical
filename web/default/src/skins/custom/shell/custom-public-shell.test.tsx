import { createElement } from 'react'
import assert from 'node:assert/strict'
import { register } from 'node:module'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'

const cssLoaderSource = `
export async function load(url, context, nextLoad) {
  if (url.endsWith('.css')) {
    return { format: 'module', shortCircuit: true, source: '' }
  }

  return nextLoad(url, context)
}
`

// Node does not load CSS side-effect imports during server-rendered tests.
register(
  `data:text/javascript,${encodeURIComponent(cssLoaderSource)}`,
  import.meta.url
)

const { CustomPublicShell } = await import('./custom-public-shell')

describe('CustomPublicShell', () => {
  it('renders its children inside the custom shell marker', () => {
    const markup = renderToStaticMarkup(
      createElement(
        CustomPublicShell,
        null,
        createElement('span', null, 'Shell content')
      )
    )

    assert.equal(
      markup,
      '<div data-skin-shell="custom"><span>Shell content</span></div>'
    )
  })
})
