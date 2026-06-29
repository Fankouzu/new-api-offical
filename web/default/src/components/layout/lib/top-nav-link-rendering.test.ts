import * as React from 'react'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'

globalThis.React = React

const { TopNavLinkIconList } = await import('./top-nav-link-rendering')

test('TopNavLinkIconList renders disabled external links without a navigable href', () => {
  const html = renderToStaticMarkup(
    React.createElement(TopNavLinkIconList, {
      links: [
        {
          title: 'Telegram',
          href: 'https://t.me/example_channel',
          external: true,
          disabled: true,
          display: 'icon',
          icon: 'telegram',
        },
      ],
    })
  )

  assert.doesNotMatch(html, /\shref=/)
  assert.match(html, /aria-disabled="true"/)
  assert.match(html, /tabindex="-1"/)
})

test('TopNavLinkIconList allows text fallback links to size to their label', () => {
  const html = renderToStaticMarkup(
    React.createElement(TopNavLinkIconList, {
      links: [
        {
          title: 'Status',
          href: '/status',
          external: true,
          display: 'text',
        },
      ],
    })
  )

  assert.doesNotMatch(html, /size-9/)
  assert.match(html, /px-3/)
  assert.match(html, />Status</)
})
