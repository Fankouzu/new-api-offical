import { beforeEach, describe, expect, test } from 'bun:test'

import { getFirstTouchAttribution } from '../src/lib/first-touch-attribution'

const values = new Map<string, string>()

const fakeStorage = {
  getItem(key: string): string | null {
    return values.get(key) ?? null
  },
  setItem(key: string, value: string): void {
    values.set(key, value)
  },
}

beforeEach(() => {
  values.clear()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: fakeStorage,
  })
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      location: {
        href: 'https://lizh.ai/wallet?utm_source=google&gclid=click-123',
        search: '?utm_source=google&gclid=click-123',
      },
    },
  })
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      cookie:
        '_ga=GA1.1.123456789.987654321; _ga_TEST=GS1.1.1740000000.1.1.1740000100.0.0.0',
      referrer: '',
    },
  })
})

describe('first-touch attribution', () => {
  test('captures the GA client and session identifiers', () => {
    expect(getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        client_id: '123456789.987654321',
        session_id: '1740000000',
      })
    )
  })

  test('refreshes identifiers that were unavailable on first page load', () => {
    localStorage.setItem(
      'lizh_first_touch_attribution',
      JSON.stringify({
        page_location: 'https://lizh.ai/wallet',
        first_visit_at: '2026-07-11T00:00:00.000Z',
      })
    )

    expect(getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        client_id: '123456789.987654321',
        session_id: '1740000000',
      })
    )
  })
})
