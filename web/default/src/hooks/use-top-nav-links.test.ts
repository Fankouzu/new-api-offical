import { mergeSkinHeaderLinks } from '@/skins/runtime/navigation'
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

test('skin navigation cannot reintroduce backend-disabled pricing', () => {
  const modules = parseHeaderNavModules(
    JSON.stringify({ pricing: { enabled: false, requireAuth: false } })
  )
  const hostLinks = buildTopNavLinks({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })

  assert.equal(
    hostLinks.some((link) => link.href === '/pricing'),
    false
  )
  const merged = mergeSkinHeaderLinks(
    hostLinks,
    [
      {
        id: 'solutions',
        path: '/solutions',
        componentImport: './solutions',
        navigation: { labelKey: 'Solutions', position: 'header' },
      },
    ],
    identity
  )

  assert.equal(
    merged.some((link) => link.href === '/pricing'),
    false
  )
})

test('declared skin route appends after About and end custom links', () => {
  const modules = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'community',
          title: 'Community',
          href: 'https://example.com/community',
          enabled: true,
          external: true,
          requireAuth: false,
          position: 'end',
        },
      ],
    })
  )
  const hostLinks = buildTopNavLinks({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })

  const merged = mergeSkinHeaderLinks(
    hostLinks,
    [
      {
        id: 'solutions',
        path: '/solutions',
        componentImport: './solutions',
        navigation: { labelKey: 'Solutions', position: 'header' },
      },
    ],
    identity
  )

  assert.deepEqual(
    merged.slice(-3).map((link) => link.title),
    ['About', 'Community', 'Solutions']
  )
})

test('merging skin navigation into primary leaves utility slots unchanged', () => {
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
  const utilityLink = slots.before_search[0]

  const merged = {
    ...slots,
    primary: mergeSkinHeaderLinks(
      slots.primary,
      [
        {
          id: 'solutions',
          path: '/solutions',
          componentImport: './solutions',
          navigation: { labelKey: 'Solutions', position: 'header' },
        },
      ],
      identity
    ),
  }

  assert.equal(merged.before_search, slots.before_search)
  assert.equal(merged.before_search[0], utilityLink)
  assert.equal(merged.primary.at(-1)?.href, '/solutions')
})

test('backend custom link wins over a duplicate skin route contribution', () => {
  const modules = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'solutions',
          title: 'Admin solutions',
          href: '/solutions',
          enabled: true,
          external: false,
          requireAuth: false,
          position: 'end',
        },
      ],
    })
  )
  const hostLinks = buildTopNavLinks({
    modules,
    docsLink: null,
    isAuthed: false,
    t: identity,
  })
  const adminLink = hostLinks.at(-1)

  const merged = mergeSkinHeaderLinks(
    hostLinks,
    [
      {
        id: 'solutions',
        path: '/solutions',
        componentImport: './solutions',
        navigation: { labelKey: 'Skin solutions', position: 'header' },
      },
    ],
    identity
  )

  assert.equal(merged.filter((link) => link.href === '/solutions').length, 1)
  assert.equal(merged.at(-1), adminLink)
  assert.equal(merged.at(-1)?.title, 'Admin solutions')
})
