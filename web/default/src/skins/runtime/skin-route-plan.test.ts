import type { ComponentType } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { ThemeManifest } from './contracts'
import { createSkinRouteRenderPlan } from './skin-route-plan'

const Route: ComponentType = () => null
const Shell: ComponentType<{ children: React.ReactNode }> = () => null

function createManifest(shell?: ThemeManifest['shell']): ThemeManifest {
  return {
    id: 'custom',
    build: {
      id: 'custom',
      routes: [
        { id: 'solutions', path: '/solutions', componentImport: './solutions' },
      ],
    },
    pages: {},
    routes: [{ id: 'solutions', path: '/solutions', component: Route }],
    shell,
  }
}

describe('createSkinRouteRenderPlan', () => {
  it('uses the active skin ID, shell, and matching route component', () => {
    const plan = createSkinRouteRenderPlan(
      createManifest(Shell),
      'solutions',
      '/solutions'
    )

    assert.equal(plan.shell, 'skin')
    assert.equal(plan.skinId, 'custom')
    assert.equal(plan.Shell, Shell)
    assert.equal(plan.definition.component, Route)
    assert.equal(plan.definition.shell, 'skin')
  })

  it('rejects a skin route when the active manifest has no shell', () => {
    assert.throws(
      () =>
        createSkinRouteRenderPlan(createManifest(), 'solutions', '/solutions'),
      {
        message:
          'Skin custom route solutions at generated path /solutions requires a skin shell, but none is configured',
      }
    )
  })
})
