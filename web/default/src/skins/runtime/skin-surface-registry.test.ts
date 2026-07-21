import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  createSkinSurfaceToken,
  registerSkinSurface,
  unregisterSkinSurface,
} from './skin-surface-registry'

type Dataset = { skinSurface?: string }

describe('skin surface registry', () => {
  it('restores the older boundary after the newer boundary unregisters', () => {
    const dataset: Dataset = {}
    const outer = createSkinSurfaceToken()
    const inner = createSkinSurfaceToken()

    registerSkinSurface(dataset, outer, 'A')
    assert.equal(dataset.skinSurface, 'A')

    registerSkinSurface(dataset, inner, 'B')
    assert.equal(dataset.skinSurface, 'B')

    unregisterSkinSurface(dataset, inner)
    assert.equal(dataset.skinSurface, 'A')

    unregisterSkinSurface(dataset, outer)
    assert.equal(dataset.skinSurface, undefined)
  })

  it('tracks separate owners that use the same skin ID', () => {
    const dataset: Dataset = {}
    const outer = createSkinSurfaceToken()
    const inner = createSkinSurfaceToken()

    registerSkinSurface(dataset, outer, 'custom')
    registerSkinSurface(dataset, inner, 'custom')
    unregisterSkinSurface(dataset, inner)

    assert.equal(dataset.skinSurface, 'custom')

    unregisterSkinSurface(dataset, outer)
    assert.equal(dataset.skinSurface, undefined)
  })

  it('makes an updated registration the active surface', () => {
    const dataset: Dataset = {}
    const outer = createSkinSurfaceToken()
    const inner = createSkinSurfaceToken()

    registerSkinSurface(dataset, outer, 'A')
    registerSkinSurface(dataset, inner, 'B')
    registerSkinSurface(dataset, outer, 'C')

    assert.equal(dataset.skinSurface, 'C')

    unregisterSkinSurface(dataset, outer)
    assert.equal(dataset.skinSurface, 'B')
  })

  it('restores remaining owners across React-style prop effect replacement', () => {
    const dataset: Dataset = {}
    const existing = createSkinSurfaceToken()
    const updated = createSkinSurfaceToken()

    registerSkinSurface(dataset, existing, 'B')
    registerSkinSurface(dataset, updated, 'A')

    unregisterSkinSurface(dataset, updated)
    assert.equal(dataset.skinSurface, 'B')

    registerSkinSurface(dataset, updated, 'C')
    assert.equal(dataset.skinSurface, 'C')

    unregisterSkinSurface(dataset, updated)
    assert.equal(dataset.skinSurface, 'B')
  })

  it('handles StrictMode-like effect replay without duplicating ownership', () => {
    const dataset: Dataset = {}
    const outer = createSkinSurfaceToken()
    const replayed = createSkinSurfaceToken()

    registerSkinSurface(dataset, outer, 'A')
    registerSkinSurface(dataset, replayed, 'B')
    registerSkinSurface(dataset, replayed, 'B')
    unregisterSkinSurface(dataset, replayed)
    unregisterSkinSurface(dataset, replayed)

    assert.equal(dataset.skinSurface, 'A')

    registerSkinSurface(dataset, replayed, 'B')
    assert.equal(dataset.skinSurface, 'B')

    unregisterSkinSurface(dataset, replayed)
    assert.equal(dataset.skinSurface, 'A')
  })

  it('isolates registrations for separate datasets', () => {
    const firstDataset: Dataset = {}
    const secondDataset: Dataset = {}
    const first = createSkinSurfaceToken()
    const second = createSkinSurfaceToken()

    registerSkinSurface(firstDataset, first, 'A')
    registerSkinSurface(secondDataset, second, 'B')
    unregisterSkinSurface(firstDataset, first)

    assert.equal(firstDataset.skinSurface, undefined)
    assert.equal(secondDataset.skinSurface, 'B')
  })
})
