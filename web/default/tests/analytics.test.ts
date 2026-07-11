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
  delete (window as Window & { __GOOGLE_ANALYTICS_ID__?: string })
    .__GOOGLE_ANALYTICS_ID__
  delete window.gtag
  delete window.dataLayer
  document.head
    .querySelectorAll('[data-google-analytics-script="true"]')
    .forEach((node) => node.remove())
})

function dataLayerAsCommands(): unknown[][] {
  return (window.dataLayer || []).map((item) =>
    Array.from(item as ArrayLike<unknown>)
  )
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

  test('removes encoded OAuth navigation targets from the initial config', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/sign-in?redirect=%2Foauth%2Fgithub%3Fcode%3Dsecret-code%26state%3Dsecret-state&next=%2Foauth%2Fgithub%3Fcode%3Dother-code%26state%3Dother-state&page=2',
      },
    })

    initGoogleAnalytics('G-TEST123')

    expect(dataLayerAsCommands()).toEqual([
      ['js', expect.any(Date)],
      [
        'config',
        'G-TEST123',
        {
          page_location: 'http://localhost/sign-in?page=2',
        },
      ],
    ])
  })

  test('preserves repeated question marks in the initial page view', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/usage-logs/common?page=2?page=2',
      },
    })

    initGoogleAnalytics('G-TEST123')

    expect(dataLayerAsCommands()).toEqual([
      ['js', expect.any(Date)],
      [
        'config',
        'G-TEST123',
        {
          page_location: 'http://localhost/usage-logs/common?page=2?page=2',
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

  test('uses the previous tracked page as the next page referrer', () => {
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value:
        'https://search.example.test/results?utm_source=search&token=secret',
    })
    initGoogleAnalytics('G-TEST123')

    trackPageView('/pricing')
    trackPageView('/wallet')

    expect(dataLayerAsCommands().slice(-2)).toEqual([
      [
        'event',
        'page_view',
        {
          page_path: '/pricing',
          page_location: 'http://localhost/pricing',
          page_referrer:
            'https://search.example.test/results?utm_source=search',
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
      [
        'event',
        'page_view',
        {
          page_path: '/wallet',
          page_location: 'http://localhost/wallet',
          page_referrer: 'http://localhost/pricing',
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
    ])
  })

  test('uses the automatic initial page as the first explicit referrer', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/user/reset?token=secret&utm_source=email',
      },
    })
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value: 'https://search.example.test/results?utm_source=search',
    })

    initGoogleAnalytics('G-TEST123')
    trackPageView('/pricing')

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/pricing',
        page_location: 'http://localhost/pricing',
        page_referrer: 'http://localhost/user/reset?utm_source=email',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('does not record the initial page when config throws', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/landing',
      },
    })
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value: 'https://search.example.test/results?utm_source=search',
    })
    Object.defineProperty(window, 'dataLayer', {
      configurable: true,
      writable: true,
      value: {
        push(item: ArrayLike<unknown>): number {
          if (Array.from(item)[0] === 'config') {
            throw new Error('gtag config failed')
          }
          return 1
        },
      },
    })

    expect(() => initGoogleAnalytics('G-TEST123')).toThrow('gtag config failed')

    const sentCommands: unknown[][] = []
    window.gtag = (...args: unknown[]) => {
      sentCommands.push(args)
    }
    trackPageView('/pricing')

    expect(sentCommands.at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/pricing',
        page_location: 'http://localhost/pricing',
        page_referrer: 'https://search.example.test/results?utm_source=search',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('reset clears the previously tracked page location', () => {
    initGoogleAnalytics('G-TEST123')
    trackPageView('/pricing')

    resetAnalyticsForTests()
    Object.defineProperty(document, 'referrer', {
      configurable: true,
      value:
        'https://search.example.test/results?utm_source=search&token=secret',
    })
    initGoogleAnalytics('G-TEST123')
    trackPageView('/wallet')

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/wallet',
        page_location: 'http://localhost/wallet',
        page_referrer: 'https://search.example.test/results?utm_source=search',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('does not update the previous page when gtag rejects a page view', () => {
    initGoogleAnalytics('G-TEST123')
    const sentCommands: unknown[][] = []
    let rejectNextPageView = true
    window.gtag = (...args: unknown[]) => {
      if (
        args[0] === 'event' &&
        args[1] === 'page_view' &&
        rejectNextPageView
      ) {
        rejectNextPageView = false
        throw new Error('gtag send failed')
      }
      sentCommands.push(args)
    }

    expect(() => trackPageView('/pricing')).toThrow('gtag send failed')
    trackPageView('/wallet')

    expect(sentCommands.at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/wallet',
        page_location: 'http://localhost/wallet',
        page_referrer: '',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('removes reset credentials from explicit page views', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView(
      '/user/reset?email=user%40example.com&token=secret-token&utm_source=email'
    )

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/user/reset?utm_source=email',
        page_location: 'http://localhost/user/reset?utm_source=email',
        page_referrer: '',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('does not report undefined or null route values as page paths', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('undefined')
    trackPageView('/undefined')
    trackPageView('null')

    expect(dataLayerAsCommands().slice(2)).toEqual([])
  })

  test('preserves repeated question marks when reporting page views', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('/usage-logs/common?page=2?page=2')

    expect(dataLayerAsCommands().slice(2)).toEqual([
      [
        'event',
        'page_view',
        {
          page_path: '/usage-logs/common?page=2?page=2',
          page_location: 'http://localhost/usage-logs/common?page=2?page=2',
          page_referrer: '',
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
    ])
  })

  test.each([
    '/usage-logs/common?page=2?page=2',
    '/search?q=foo?q=foo',
    '/callback?next=https://example.com/path?next=https://example.com/path',
  ])('normalization preserves the query in %s', (path) => {
    expect(normalizeAnalyticsPagePath(path)).toBe(path)
  })

  test('normalizes nested URLs without exposing next in page views', () => {
    const path =
      '/callback?next=https://example.com/path?next=https://example.com/path'

    initGoogleAnalytics('G-TEST123')
    trackPageView(path)

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/callback',
        page_location: 'http://localhost/callback',
        page_referrer: '',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test('preserves query parameter order and encoding in page views', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        origin: 'http://localhost',
        href: 'http://localhost/search?gclid=a%20b&utm_source=x&utm_source=y',
      },
    })
    initGoogleAnalytics('G-TEST123')

    trackPageView('/search?gclid=a%20b&utm_source=x&utm_source=y')

    const expectedLocation =
      'http://localhost/search?gclid=a%20b&utm_source=x&utm_source=y'
    expect(dataLayerAsCommands()).toEqual([
      ['js', expect.any(Date)],
      [
        'config',
        'G-TEST123',
        {
          page_location: expectedLocation,
        },
      ],
      [
        'event',
        'page_view',
        {
          page_path: '/search?gclid=a%20b&utm_source=x&utm_source=y',
          page_location: expectedLocation,
          page_referrer: expectedLocation,
          hostname: 'localhost',
          page_title: document.title,
        },
      ],
    ])
  })

  test('removes token values from usage log page views', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('/usage-logs/common?token=production-key&page=2')

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/usage-logs/common?page=2',
        page_location: 'http://localhost/usage-logs/common?page=2',
        page_referrer: '',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
  })

  test.each(['redirect', 'next', 'return_url', 'return_to'])(
    'removes the %s navigation target from explicit page views',
    (key) => {
      initGoogleAnalytics('G-TEST123')

      trackPageView(
        `/sign-in?${key}=%2Foauth%2Fgithub%3Fcode%3Dsecret-code%26state%3Dsecret-state`
      )

      expect(dataLayerAsCommands().at(-1)).toEqual([
        'event',
        'page_view',
        {
          page_path: '/sign-in',
          page_location: 'http://localhost/sign-in',
          page_referrer: '',
          hostname: 'localhost',
          page_title: document.title,
        },
      ])
    }
  )

  test('removes OAuth callback credentials from page views', () => {
    initGoogleAnalytics('G-TEST123')

    trackPageView('/oauth/github?code=secret-code&state=secret-state')

    expect(dataLayerAsCommands().at(-1)).toEqual([
      'event',
      'page_view',
      {
        page_path: '/oauth/github',
        page_location: 'http://localhost/oauth/github',
        page_referrer: '',
        hostname: 'localhost',
        page_title: document.title,
      },
    ])
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
