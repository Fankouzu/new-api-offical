import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { SkinRouteBuildDefinition } from './build-contracts'
import { mergeSkinFooterLinks, mergeSkinHeaderLinks } from './navigation'

const identity = (value: string) => value

function route(
  path: `/${string}`,
  navigation?: SkinRouteBuildDefinition['navigation']
): SkinRouteBuildDefinition {
  return { id: path.slice(1), path, componentImport: `.${path}`, navigation }
}

test('mergeSkinHeaderLinks derives, sorts, and translates header links without mutation', () => {
  const hostLink = Object.freeze({
    title: 'Host',
    href: '/host',
    disabled: true,
    external: true,
    display: 'icon' as const,
    icon: 'host-icon',
  })
  const hostLinks = Object.freeze([hostLink])
  const routes = Object.freeze([
    Object.freeze(
      route('/zoo', { labelKey: 'Zoo', position: 'header', order: 20 })
    ),
    Object.freeze(
      route('/alpha', { labelKey: 'Alpha', position: 'header', order: 10 })
    ),
    Object.freeze(
      route('/beta', { labelKey: 'Beta', position: 'header', order: 10 })
    ),
    Object.freeze(
      route('/footer', { labelKey: 'Footer', position: 'footer', order: 0 })
    ),
  ])

  const result = mergeSkinHeaderLinks(
    hostLinks,
    routes,
    (key) => `translated:${key}`
  )

  assert.deepEqual(result, [
    hostLink,
    { title: 'translated:Alpha', href: '/alpha' },
    { title: 'translated:Beta', href: '/beta' },
    { title: 'translated:Zoo', href: '/zoo' },
  ])
  assert.notEqual(result, hostLinks)
  assert.equal(result[0], hostLink)
  assert.deepEqual(
    routes.map((item) => item.path),
    ['/zoo', '/alpha', '/beta', '/footer']
  )
})

test('mergeSkinHeaderLinks sorts absent orders after ordered links by href', () => {
  const result = mergeSkinHeaderLinks(
    [],
    [
      route('/c', { labelKey: 'C', position: 'header' }),
      route('/b', { labelKey: 'B', position: 'header', order: 1 }),
      route('/a', { labelKey: 'A', position: 'header', order: 1 }),
    ],
    identity
  )

  assert.deepEqual(
    result.map((link) => link.href),
    ['/a', '/b', '/c']
  )
})

test('mergeSkinHeaderLinks lets an existing host link win without throwing', () => {
  const hostLink = Object.freeze({
    title: 'Admin solutions',
    href: '/solutions',
    external: false,
  })

  const result = mergeSkinHeaderLinks(
    [hostLink],
    [route('/solutions', { labelKey: 'Skin solutions', position: 'header' })],
    identity
  )

  assert.deepEqual(result, [hostLink])
  assert.equal(result[0], hostLink)
})

test('mergeSkinHeaderLinks cannot create pricing without a pricing build route', () => {
  const result = mergeSkinHeaderLinks(
    [],
    [route('/solutions', { labelKey: 'Solutions', position: 'header' })],
    identity
  )

  assert.equal(
    result.some((link) => link.href === '/pricing'),
    false
  )
})

test('mergeSkinHeaderLinks skips unexpected invalid runtime metadata', () => {
  const invalidRoutes = [
    route('/empty', { labelKey: '', position: 'header' }),
    route('/position', {
      labelKey: 'Position',
      position: 'sidebar',
    } as never),
    route('/order', {
      labelKey: 'Order',
      position: 'header',
      order: Number.NaN,
    }),
    { ...route('/missing'), navigation: { labelKey: 'Missing' } } as never,
  ]

  assert.doesNotThrow(() => mergeSkinHeaderLinks([], invalidRoutes, identity))
  assert.deepEqual(mergeSkinHeaderLinks([], invalidRoutes, identity), [])
})

test('mergeSkinHeaderLinks returns equal content in a new array without header routes', () => {
  const hostLink = { title: 'Home', href: '/', disabled: true }
  const hostLinks = [hostLink]

  const result = mergeSkinHeaderLinks(
    hostLinks,
    [route('/footer', { labelKey: 'Footer', position: 'footer' })],
    identity
  )

  assert.deepEqual(result, hostLinks)
  assert.notEqual(result, hostLinks)
  assert.equal(result[0], hostLink)
})

test('mergeSkinFooterLinks sorts contributions and omits host collisions', () => {
  const result = mergeSkinFooterLinks(
    [{ text: 'Host', href: '/host' }],
    [
      route('/later', { labelKey: 'Later', position: 'footer', order: 20 }),
      route('/host', { labelKey: 'Skin host', position: 'footer', order: 1 }),
      route('/first', { labelKey: 'First', position: 'footer', order: 1 }),
    ],
    identity
  )

  assert.deepEqual(result, [
    { text: 'Host', href: '/host' },
    { text: 'First', href: '/first' },
    { text: 'Later', href: '/later' },
  ])
})
