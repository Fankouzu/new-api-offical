import { afterEach, beforeEach, describe, expect, test } from 'bun:test'

import { syncRouteSEO } from '../src/lib/seo'

describe('client SEO route metadata', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: {
        location: { origin: 'https://lizh.ai' },
        localStorage: {
          getItem(key: string) {
            if (key === 'status') return JSON.stringify({ system_name: 'Lizh Console' })
            return null
          },
        },
      },
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        title: '',
        head: {
          querySelector() {
            return null
          },
          appendChild() {
            return undefined
          },
        },
        createElement(tagName: string) {
          return { tagName, setAttribute() {}, remove() {} }
        },
      },
    })
  })

  afterEach(() => {
    Reflect.deleteProperty(globalThis, 'window')
    Reflect.deleteProperty(globalThis, 'document')
  })

  test.each([
    '/dashboard',
    '/models',
    '/models/pricing',
    '/system-settings',
    '/system-settings/site/general',
    '/chat/abc123',
    '/keys',
    '/wallet',
  ])('does not override authenticated app route title for %s', (path) => {
    document.title = 'Localized Console Title'

    syncRouteSEO(path)

    expect(document.title).toBe('Localized Console Title')
  })

  test('restores cached system title when a legacy SEO fallback title is present', () => {
    document.title = 'Page Not Found | Lizh AI'

    syncRouteSEO('/dashboard')

    expect(document.title).toBe('Lizh Console')
  })
})
