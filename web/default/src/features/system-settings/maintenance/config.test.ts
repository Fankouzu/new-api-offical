import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parseHeaderNavModules, serializeHeaderNavModules } from './config'

test('parseHeaderNavModules keeps enabled custom external links', () => {
  const config = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'telegram',
          title: 'Telegram',
          href: 'https://t.me/example_channel',
          enabled: true,
          external: true,
          requireAuth: false,
          position: 'after_docs',
          display: 'icon',
          icon: 'telegram',
        },
      ],
    })
  )

  assert.equal(config.customLinks.length, 1)
  assert.deepEqual(config.customLinks[0], {
    id: 'telegram',
    title: 'Telegram',
    href: 'https://t.me/example_channel',
    enabled: true,
    external: true,
    requireAuth: false,
    position: 'after_docs',
    display: 'icon',
    icon: 'telegram',
  })
})

test('parseHeaderNavModules filters custom links without safe absolute or local URLs', () => {
  const config = parseHeaderNavModules(
    JSON.stringify({
      customLinks: [
        {
          id: 'bad-js',
          title: 'Bad',
          href: 'javascript:alert(1)',
          enabled: true,
        },
        {
          id: 'bad-empty',
          title: 'Empty',
          href: '',
          enabled: true,
        },
        {
          id: 'local-status',
          title: 'Status',
          href: '/status',
          enabled: true,
        },
      ],
    })
  )

  assert.deepEqual(
    config.customLinks.map((link) => link.id),
    ['local-status']
  )
})

test('serializeHeaderNavModules preserves custom links', () => {
  const serialized = serializeHeaderNavModules({
    home: true,
    console: true,
    pricing: { enabled: true, requireAuth: false },
    rankings: { enabled: true, requireAuth: false },
    docs: true,
    about: true,
    customLinks: [
      {
        id: 'telegram',
        title: 'Telegram',
        href: 'https://t.me/example_channel',
        enabled: true,
        external: true,
        requireAuth: false,
        position: 'after_docs',
        display: 'icon',
        icon: 'telegram',
      },
    ],
  })

  assert.deepEqual(JSON.parse(serialized).customLinks, [
    {
      id: 'telegram',
      title: 'Telegram',
      href: 'https://t.me/example_channel',
      enabled: true,
      external: true,
      requireAuth: false,
      position: 'after_docs',
      display: 'icon',
      icon: 'telegram',
    },
  ])
})
