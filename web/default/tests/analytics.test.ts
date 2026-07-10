import { afterEach, beforeEach, describe, expect, test } from 'bun:test'

import {
  getGoogleAnalyticsMeasurementId,
  initGoogleAnalytics,
  normalizeAnalyticsPagePath,
  resetAnalyticsForTests,
  trackAnalyticsEvent,
  trackPageView,
} from '../src/lib/analytics'

declare global {
  interface Window {
    dataLayer?: unknown[]
    gtag?: (...args: unknown[]) => void
  }
}

class FakeElement {
  async = false
  dataset: Record<string, string> = {}
  src = ''

  constructor(public readonly tagName: string) {}

  getAttribute(name: string): string | null {
    if (name === 'src') return this.src
    if (name === 'data-google-analytics-script') {
      return this.dataset.googleAnalyticsScript || null
    }
    return null
  }

  remove(): void {
    fakeHead.children = fakeHead.children.filter((child) => child !== this)
  }
}

const fakeHead = {
  children: [] as FakeElement[],
  appendChild(node: FakeElement): FakeElement {
    this.children.push(node)
    return node
  },
  querySelector(selector: string): FakeElement | null {
    if (
      selector === '[data-google-analytics-script="true"]' ||
      selector === 'script[data-google-analytics-script="true"]'
    ) {
      return (
        this.children.find(
          (child) => child.dataset.googleAnalyticsScript === 'true'
        ) || null
      )
    }
    return null
  },
  querySelectorAll(selector: string): FakeElement[] {
    const item = this.querySelector(selector)
    return item ? [item] : []
  },
}

beforeEach(() => {
  fakeHead.children = []
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      location: { origin: 'http://localhost' },
    },
  })
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      head: fakeHead,
      title: 'Test title',
      referrer: '',
      createElement: (tagName: string) => new FakeElement(tagName),
      querySelector: (selector: string) => fakeHead.querySelector(selector),
    },
  })
})

afterEach(() => {
  resetAnalyticsForTests()
  delete (
    window as Window & { __GOOGLE_ANALYTICS_ID__?: string }
  ).__GOOGLE_ANALYTICS_ID__
  delete window.gtag
  delete window.dataLayer
  document.head
    .querySelectorAll('[data-google-analytics-script="true"]')
    .forEach((node) => node.remove())
})

function dataLayerAsCommands(): unknown[][] {
  return (window.dataLayer || []).map((item) => Array.from(item as ArrayLike<unknown>))
}

describe('google analytics runtime', () => {
  test('does not hardcode a measurement id when env is not set', () => {
    expect(getGoogleAnalyticsMeasurementId()).toBe('')
  })

  test('prefers a runtime measurement id for container deployments', () => {
    ;(
      window as Window & { __GOOGLE_ANALYTICS_ID__?: string }
    ).__GOOGLE_ANALYTICS_ID__ = 'G-RUNTIME123'

    expect(getGoogleAnalyticsMeasurementId()).toBe('G-RUNTIME123')
  })

  test('does not initialize without a measurement id', () => {
    initGoogleAnalytics('')

    expect(window.gtag).toBeUndefined()
    expect(
      document.head.querySelector('[data-google-analytics-script="true"]')
    ).toBeNull()
  })

  test('injects gtag script and lets GA4 send the initial page_view', () => {
    initGoogleAnalytics('G-TEST123')

    const script = document.head.querySelector(
      'script[data-google-analytics-script="true"]'
    )
    expect(script?.getAttribute('src')).toBe(
      'https://www.googletagmanager.com/gtag/js?id=G-TEST123'
    )
    expect(dataLayerAsCommands()).toEqual([
      ['js', expect.any(Date)],
      ['config', 'G-TEST123'],
    ])
  })

  test('sanitizes the automatic initial page-view context', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/user/reset?email=user%40example.com&token=secret',
      },
    })
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value:
        'https://mail.example.test/open?email=user%40example.com&token=secret',
    })

    initGoogleAnalytics('G-TEST123')

    expect(dataLayerAsCommands()).toEqual([
      ['js', expect.any(Date)],
      [
        'config',
        'G-TEST123',
        {
          page_location: 'http://localhost/user/reset',
          page_referrer: 'https://mail.example.test/open',
        },
      ],
    ])
  })

  test('tracks page views and events after initialization', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('/pricing?model=gpt')
    trackAnalyticsEvent('sign_up_click', { method: 'oauth' })

    expect(dataLayerAsCommands().slice(2)).toEqual([
      [
        'event',
        'page_view',
        {
          page_path: '/pricing?model=gpt',
          page_location: 'http://localhost/pricing?model=gpt',
          page_referrer: '',
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
      ['event', 'sign_up_click', { method: 'oauth' }],
    ])
  })

  test('does not report undefined or null route values as page paths', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('undefined')
    trackPageView('/undefined')
    trackPageView('null')

    expect(dataLayerAsCommands().slice(2)).toEqual([])
  })

  test('normalizes duplicated question marks before reporting page views', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('/usage-logs/common?page=2?page=2')

    expect(dataLayerAsCommands().slice(2)).toEqual([
      [
        'event',
        'page_view',
        {
          page_path: '/usage-logs/common?page=2',
          page_location: 'http://localhost/usage-logs/common?page=2',
          page_referrer: '',
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
    ])
  })

  test('preserves nested URL values and repeated query keys', () => {
    expect(
      normalizeAnalyticsPagePath(
        '/callback?next=https://example.com/a?x=1&tag=a&tag=b'
      )
    ).toBe('/callback?next=https://example.com/a?x=1&tag=a&tag=b')
  })

  test('removes reset credentials from page referrers', () => {
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value:
        'https://lizh.ai/user/reset?email=user%40example.com&token=secret-token',
    })
    initGoogleAnalytics('G-TEST123')

    trackPageView('/wallet')

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/wallet',
        page_location: 'http://localhost/wallet',
        page_referrer: 'https://lizh.ai/user/reset',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })
})
