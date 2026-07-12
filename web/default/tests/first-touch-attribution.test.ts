import { beforeEach, describe, expect, test } from 'bun:test'
import * as firstTouchAttribution from '../src/lib/first-touch-attribution'

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
      __GOOGLE_ANALYTICS_ID__: 'G-TEST',
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
    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
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

    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        client_id: '123456789.987654321',
        session_id: '1740000000',
      })
    )
  })

  test('refreshes the session identifier after GA session rollover', () => {
    localStorage.setItem(
      'lizh_first_touch_attribution',
      JSON.stringify({
        client_id: '123456789.987654321',
        session_id: '1730000000',
        page_location: 'https://lizh.ai/wallet',
        source: 'google',
        first_visit_at: '2026-07-11T00:00:00.000Z',
      })
    )

    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        client_id: '123456789.987654321',
        session_id: '1740000000',
        source: 'google',
      })
    )
  })

  test('reads the session cookie for the configured measurement id', () => {
    Object.defineProperty(document, 'cookie', {
      configurable: true,
      value:
        '_ga=GA1.1.123456789.987654321; _ga_OTHER=GS1.1.9990000000.1.1.9990000100.0.0.0; _ga_TEST=GS1.1.1740000000.1.1.1740000100.0.0.0',
    })

    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        session_id: '1740000000',
      })
    )
  })

  test('reads the session id from the GA4 GS2 cookie format', () => {
    Object.defineProperty(document, 'cookie', {
      configurable: true,
      value:
        '_ga=GA1.1.123456789.987654321; _ga_TEST=GS2.1.s1740000001$o1$g0$t1740000100$j60$l0$h0',
    })

    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
      expect.objectContaining({
        session_id: '1740000001',
      })
    )
  })

  test('ignores malformed encoded GA cookies without throwing', () => {
    Object.defineProperty(document, 'cookie', {
      configurable: true,
      value: '_ga=%E0%A4%A; _ga_TEST=%E0%A4%A',
    })

    expect(() => firstTouchAttribution.getFirstTouchAttribution()).not.toThrow()
    expect(firstTouchAttribution.getFirstTouchAttribution()).toEqual(
      expect.not.objectContaining({
        client_id: expect.anything(),
        session_id: expect.anything(),
      })
    )
  })

  test('adds the current attribution only to an explicitly enriched request', () => {
    const enrich = (
      firstTouchAttribution as typeof firstTouchAttribution & {
        withFirstTouchAttribution?: (request: { amount: number }) => object
      }
    ).withFirstTouchAttribution
    expect(typeof enrich).toBe('function')

    expect(enrich?.({ amount: 10 })).toEqual({
      amount: 10,
      attribution: expect.objectContaining({
        client_id: '123456789.987654321',
        session_id: '1740000000',
        gclid: 'click-123',
      }),
    })
  })

  test('does not attach invalid stored attribution values', () => {
    const request = { amount: 10 }
    const invalidValues = [[], 'not-an-attribution', { source: 123 }, {}]

    for (const value of invalidValues) {
      localStorage.setItem(
        'lizh_first_touch_attribution',
        JSON.stringify(value)
      )

      expect(firstTouchAttribution.withFirstTouchAttribution(request)).toBe(
        request
      )
    }
  })

  test('does not attach a stored null attribution', () => {
    const request = { amount: 10 }
    localStorage.setItem('lizh_first_touch_attribution', 'null')

    expect(firstTouchAttribution.withFirstTouchAttribution(request)).toBe(
      request
    )
  })

  test('copies only known string fields from stored attribution', () => {
    const request = { amount: 10 }
    localStorage.setItem(
      'lizh_first_touch_attribution',
      JSON.stringify({
        source: 'google',
        unexpected_private_value: 'secret',
      })
    )

    expect(firstTouchAttribution.withFirstTouchAttribution(request)).toEqual({
      amount: 10,
      attribution: {
        source: 'google',
        client_id: '123456789.987654321',
        session_id: '1740000000',
      },
    })
  })
})
