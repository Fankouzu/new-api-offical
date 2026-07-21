import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { ThemeManifest } from './contracts'
import { resolveRuntimeRoute } from './resolve-route'

const First: ComponentType = () => null
const Second: ComponentType = () => null

describe('resolveRuntimeRoute', () => {
  function createManifest(
    buildRoutes: ThemeManifest['build']['routes'],
    runtimeRoutes: ThemeManifest['routes']
  ): ThemeManifest {
    return {
      id: 'custom',
      build: { id: 'custom', routes: buildRoutes },
      pages: {},
      routes: runtimeRoutes,
    }
  }

  it('returns the component registered for the exact route ID', () => {
    const manifest = createManifest(
      [
        { id: 'first', path: '/first', componentImport: './first' },
        { id: 'second', path: '/second', componentImport: './second' },
      ],
      [
        { id: 'first', path: '/first', component: First },
        { id: 'second', path: '/second', component: Second },
      ]
    )

    assert.equal(resolveRuntimeRoute(manifest, 'second', '/second'), Second)
  })

  it('rejects a route missing from the build manifest', () => {
    assert.throws(
      () => resolveRuntimeRoute(createManifest([], []), 'missing', '/new'),
      {
        message:
          'Skin custom route missing at generated path /new is missing from the build manifest',
      }
    )
  })

  it('rejects duplicate build registrations', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          createManifest(
            [
              { id: 'duplicate', path: '/first', componentImport: './first' },
              { id: 'duplicate', path: '/second', componentImport: './second' },
            ],
            []
          ),
          'duplicate',
          '/generated'
        ),
      {
        message:
          'Skin custom route duplicate at generated path /generated is registered more than once in the build manifest',
      }
    )
  })

  it('rejects a build path that differs from the generated proxy', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          createManifest(
            [{ id: 'solutions', path: '/old', componentImport: './solutions' }],
            [{ id: 'solutions', path: '/solutions', component: First }]
          ),
          'solutions',
          '/solutions'
        ),
      {
        message:
          'Skin custom route solutions build path mismatch for generated path /solutions: actual path /old',
      }
    )
  })

  it('rejects a route missing from the runtime manifest', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          createManifest(
            [
              {
                id: 'solutions',
                path: '/solutions',
                componentImport: './solutions',
              },
            ],
            []
          ),
          'solutions',
          '/solutions'
        ),
      {
        message:
          'Skin custom route solutions at generated path /solutions is missing from the runtime manifest',
      }
    )
  })

  it('rejects duplicate runtime registrations', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          createManifest(
            [
              {
                id: 'solutions',
                path: '/solutions',
                componentImport: './solutions',
              },
            ],
            [
              { id: 'solutions', path: '/solutions', component: First },
              { id: 'solutions', path: '/other', component: Second },
            ]
          ),
          'solutions',
          '/solutions'
        ),
      {
        message:
          'Skin custom route solutions at generated path /solutions is registered more than once in the runtime manifest',
      }
    )
  })

  it('rejects a runtime path that differs from the generated proxy', () => {
    assert.throws(
      () =>
        resolveRuntimeRoute(
          createManifest(
            [
              {
                id: 'solutions',
                path: '/solutions',
                componentImport: './solutions',
              },
            ],
            [{ id: 'solutions', path: '/old', component: First }]
          ),
          'solutions',
          '/solutions'
        ),
      {
        message:
          'Skin custom route solutions runtime path mismatch for generated path /solutions: actual path /old',
      }
    )
  })

  it('never renders the same ID through a stale different-path proxy', () => {
    const manifest = createManifest(
      [{ id: 'solutions', path: '/new', componentImport: './solutions' }],
      [{ id: 'solutions', path: '/new', component: First }]
    )

    assert.throws(
      () => resolveRuntimeRoute(manifest, 'solutions', '/old'),
      /actual path \/new/
    )
  })
})
