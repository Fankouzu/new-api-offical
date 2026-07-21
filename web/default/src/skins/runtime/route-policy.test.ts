import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { assertSkinRouteAllowed, getPublicPagePath } from './route-policy'

describe('skin route policy', () => {
  it('maps public page keys to host paths', () => {
    assert.deepEqual(
      {
        home: getPublicPagePath('home'),
        pricing: getPublicPagePath('pricing'),
        modelDetails: getPublicPagePath('modelDetails'),
        rankings: getPublicPagePath('rankings'),
        about: getPublicPagePath('about'),
        privacyPolicy: getPublicPagePath('privacyPolicy'),
        userAgreement: getPublicPagePath('userAgreement'),
      },
      {
        home: '/',
        pricing: '/pricing',
        modelDetails: '/pricing/$modelId',
        rankings: '/rankings',
        about: '/about',
        privacyPolicy: '/privacy-policy',
        userAgreement: '/user-agreement',
      }
    )
  })

  it('rejects host-owned and protected paths', () => {
    for (const path of [
      '/',
      '/pricing',
      '/sign-in',
      '/setup',
      '/500',
      '/dashboard',
    ]) {
      assert.throws(() => assertSkinRouteAllowed(path))
    }
  })

  it('accepts a new public skin route', () => {
    assert.doesNotThrow(() => assertSkinRouteAllowed('/guides/$slug'))
  })
})
