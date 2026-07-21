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

  it('rejects invalid, host-owned, and protected paths', () => {
    for (const path of [
      'guides',
      '/guides//draft',
      '/guides?draft=true',
      '/guides#draft',
      '/pricing/',
      '/pr%69cing',
      '/guides\\draft',
      '/guides/\u0000draft',
      '/guides/./draft',
      '/guides/../draft',
      '/',
      '/pricing',
      '/sign-in',
      '/setup',
      '/500',
      '/dashboard',
      '/chat2link',
      '/errors',
      '/errors/500',
    ]) {
      assert.throws(() => assertSkinRouteAllowed(path), path)
    }
  })

  it('accepts new public skin routes outside protected prefix boundaries', () => {
    for (const path of ['/guides/$slug', '/dashboard-guide']) {
      assert.doesNotThrow(() => assertSkinRouteAllowed(path), path)
    }
  })
})
