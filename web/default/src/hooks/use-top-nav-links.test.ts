import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parseHeaderNavModules } from '@/features/system-settings/maintenance/config'
import { buildTopNavLinks } from './use-top-nav-links'

const identity = (value: string) => value

test('buildTopNavLinks places end custom links after About', () => {
  const modules = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'telegram',
          title: 'Telegram',
          href: 'https://t.me/example_channel',
          enabled: true,
          external: true,
          requireAuth: false,
          position: 'end',
          icon: 'telegram',
          display: 'icon',
        },
      ],
    })
  )

  const links = buildTopNavLinks({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })

  assert.deepEqual(
    links.map((link) => link.title),
    ['Home', 'Console', 'Model Square', 'Rankings', 'Docs', 'About', 'Telegram']
  )
})

test('buildTopNavLinks preserves icon display metadata for social links', () => {
  const modules = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'telegram',
          title: 'Telegram',
          href: 'https://t.me/example_channel',
          enabled: true,
          external: true,
          requireAuth: false,
          position: 'end',
          icon: 'telegram',
          display: 'icon',
        },
      ],
    })
  )

  const links = buildTopNavLinks({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })

  const telegram = links.find((link) => link.title === 'Telegram')

  assert.equal(telegram?.display, 'icon')
  assert.equal(telegram?.icon, 'telegram')
})
