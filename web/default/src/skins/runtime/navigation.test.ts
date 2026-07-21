import assert from 'node:assert/strict'
import { test } from 'node:test'
import { mergeSkinHeaderLinks } from './navigation'

const identity = (value: string) => value

test('mergeSkinHeaderLinks appends sorted translated skin links without mutating inputs', () => {
  const hostLink = Object.freeze({
    title: 'Host',
    href: '/host',
    disabled: true,
    external: true,
    display: 'icon' as const,
    icon: 'host-icon',
  })
  const hostLinks = Object.freeze([hostLink])
  const contributions = Object.freeze([
    Object.freeze({ labelKey: 'Zoo', href: '/zoo' as const, order: 20 }),
    Object.freeze({ labelKey: 'Alpha', href: '/alpha' as const, order: 10 }),
    Object.freeze({ labelKey: 'Beta', href: '/beta' as const, order: 10 }),
  ])
  const allowedPaths = Object.freeze(['/zoo', '/alpha', '/beta'] as const)

  const result = mergeSkinHeaderLinks(
    hostLinks,
    contributions,
    allowedPaths,
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
    contributions.map((link) => link.href),
    ['/zoo', '/alpha', '/beta']
  )
})

test('mergeSkinHeaderLinks sorts equal and absent orders by href', () => {
  const result = mergeSkinHeaderLinks(
    [],
    [
      { labelKey: 'C', href: '/c' },
      { labelKey: 'B', href: '/b', order: 1 },
      { labelKey: 'A', href: '/a', order: 1 },
    ],
    ['/a', '/b', '/c'],
    identity
  )

  assert.deepEqual(
    result.map((link) => link.href),
    ['/a', '/b', '/c']
  )
})

test('mergeSkinHeaderLinks rejects duplicate contribution hrefs', () => {
  assert.throws(
    () =>
      mergeSkinHeaderLinks(
        [],
        [
          { labelKey: 'First', href: '/solutions' },
          { labelKey: 'Second', href: '/solutions' },
        ],
        ['/solutions'],
        identity
      ),
    /duplicate.*\/solutions/i
  )
})

test('mergeSkinHeaderLinks rejects paths absent from active build routes', () => {
  assert.throws(
    () =>
      mergeSkinHeaderLinks(
        [],
        [{ labelKey: 'Pricing', href: '/pricing' }],
        ['/solutions'],
        identity
      ),
    /\/pricing/
  )
})

test('mergeSkinHeaderLinks applies route policy even when a protected path is listed', () => {
  assert.throws(
    () =>
      mergeSkinHeaderLinks(
        [],
        [{ labelKey: 'Dashboard', href: '/dashboard' }],
        ['/dashboard'],
        identity
      ),
    /\/dashboard/
  )
})

test('mergeSkinHeaderLinks rejects a contribution already present in host links', () => {
  assert.throws(
    () =>
      mergeSkinHeaderLinks(
        [{ title: 'Host solutions', href: '/solutions' }],
        [{ labelKey: 'Skin solutions', href: '/solutions' }],
        ['/solutions'],
        identity
      ),
    /host.*\/solutions/i
  )
})

test('mergeSkinHeaderLinks returns equal content in a new array when contributions are empty', () => {
  const hostLink = { title: 'Home', href: '/', disabled: true }
  const hostLinks = [hostLink]

  const result = mergeSkinHeaderLinks(hostLinks, [], [], identity)

  assert.deepEqual(result, hostLinks)
  assert.notEqual(result, hostLinks)
  assert.equal(result[0], hostLink)
})
