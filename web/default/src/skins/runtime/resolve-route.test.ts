import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { SkinRuntimeRoute } from './contracts'
import { resolveRuntimeRoute } from './resolve-route'

const First: ComponentType = () => null
const Second: ComponentType = () => null

describe('resolveRuntimeRoute', () => {
  it('returns the component registered for the exact route ID', () => {
    const routes: readonly SkinRuntimeRoute[] = [
      { id: 'first', component: First },
      { id: 'second', component: Second },
    ]

    assert.equal(resolveRuntimeRoute(routes, 'second'), Second)
  })

  it('rejects an unregistered route ID with an actionable error', () => {
    assert.throws(() => resolveRuntimeRoute([], 'missing'), {
      message: 'Active skin route is not registered: missing',
    })
  })

  it('rejects duplicate runtime registrations as ambiguous', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          [
            { id: 'duplicate', component: First },
            { id: 'duplicate', component: Second },
          ],
          'duplicate'
        ),
      { message: 'Active skin route is registered more than once: duplicate' }
    )
  })
})
