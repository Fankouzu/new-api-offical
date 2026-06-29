import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parseHeaderNavModules } from '@/features/system-settings/maintenance/config'
import { buildTopNavLinks, buildTopNavLinkSlots } from './use-top-nav-links'

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

test('buildTopNavLinkSlots separates utility-position custom links from primary nav', () => {
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
          position: 'before_search',
          icon: 'telegram',
          display: 'icon',
        },
      ],
    })
  )

  const slots = buildTopNavLinkSlots({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })

  assert.equal(
    slots.primary.some((link) => link.title === 'Telegram'),
    false
  )
  assert.equal(slots.before_search[0]?.title, 'Telegram')
  assert.equal(slots.before_search[0]?.display, 'icon')
})
