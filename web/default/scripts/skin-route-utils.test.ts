import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  normalizeSkinId,
  renderActiveBuildModule,
  renderActiveSkinModule,
} from './skin-route-utils'

describe('skin build selection utilities', () => {
  it('defaults missing and empty skin IDs to default', () => {
    assert.equal(normalizeSkinId(undefined), 'default')
    assert.equal(normalizeSkinId(''), 'default')
  })

  it('rejects skin IDs that cannot be used as path segments', () => {
    assert.throws(() => normalizeSkinId('../custom'))
    assert.throws(() => normalizeSkinId('custom skin'))
  })

  it('renders the selected runtime manifest as the activeSkin default export', () => {
    const source = renderActiveSkinModule('custom')

    assert.match(source, /@\/skins\/custom\/manifest/)
    assert.match(source, /export \{ default as activeSkin \}/)
    assert.doesNotMatch(source, /customSkin/)
  })

  it('renders the selected build manifest as the activeSkinBuild default export', () => {
    const source = renderActiveBuildModule('custom')

    assert.match(source, /@\/skins\/custom\/build-manifest/)
    assert.match(source, /export \{ default as activeSkinBuild \}/)
    assert.doesNotMatch(source, /customSkinBuild/)
  })
})
