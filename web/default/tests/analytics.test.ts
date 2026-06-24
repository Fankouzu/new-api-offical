import { afterEach, beforeEach, describe, expect, test } from 'bun:test'

import {
  getGoogleAnalyticsMeasurementId,
  initGoogleAnalytics,
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
      createElement: (tagName: string) => new FakeElement(tagName),
      querySelector: (selector: string) => fakeHead.querySelector(selector),
    },
  })
})

afterEach(() => {
  resetAnalyticsForTests()
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
  test('uses the repository default measurement id when env is not set', () => {
    expect(getGoogleAnalyticsMeasurementId()).toBe('G-9693VBP1VM')
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
          page_title: document.title,
        },
      ],
      ['event', 'sign_up_click', { method: 'oauth' }],
    ])
  })
})
