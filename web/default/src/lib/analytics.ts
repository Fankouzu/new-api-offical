/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { sanitizeAttributionURL } from './first-touch-attribution'

type GtagCommand = [command: string, ...args: unknown[]]
type GtagDataLayerItem = GtagCommand | IArguments

declare global {
  interface Window {
    dataLayer?: GtagDataLayerItem[]
    gtag?: (...args: GtagCommand) => void
    __GOOGLE_ANALYTICS_ID__?: string
  }
}

let activeMeasurementId = ''
let initialized = false

export function getGoogleAnalyticsMeasurementId(): string {
  const runtimeMeasurementId =
    typeof window !== 'undefined' ? window.__GOOGLE_ANALYTICS_ID__ : ''
  return (
    runtimeMeasurementId ||
    import.meta.env?.VITE_GOOGLE_ANALYTICS_ID ||
    ''
  ).trim()
}

export function initConfiguredGoogleAnalytics(): void {
  initGoogleAnalytics(getGoogleAnalyticsMeasurementId())
}

export function initGoogleAnalytics(measurementId: string): void {
  const normalizedId = measurementId.trim()
  if (
    normalizedId === '' ||
    typeof window === 'undefined' ||
    typeof document === 'undefined'
  ) {
    return
  }

  if (initialized && activeMeasurementId === normalizedId) return

  activeMeasurementId = normalizedId
  initialized = true

  window.dataLayer = window.dataLayer || []
  window.gtag = function gtag() {
    // Google gtag.js expects the pre-load queue to keep the snippet's arguments object.
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer?.push(arguments)
  }

  if (!document.querySelector('[data-google-analytics-script="true"]')) {
    const script = document.createElement('script')
    script.async = true
    script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(normalizedId)}`
    script.dataset.googleAnalyticsScript = 'true'
    document.head.appendChild(script)
  }

  window.gtag('js', new Date())
  const pageLocation =
    typeof window.location?.href === 'string'
      ? sanitizeAttributionURL(window.location.href)
      : undefined
  const pageReferrer =
    typeof document.referrer === 'string' && document.referrer
      ? sanitizeAttributionURL(document.referrer)
      : undefined
  if (pageLocation || pageReferrer) {
    window.gtag('config', normalizedId, {
      ...(pageLocation ? { page_location: pageLocation } : {}),
      ...(pageReferrer ? { page_referrer: pageReferrer } : {}),
    })
  } else {
    window.gtag('config', normalizedId)
  }
}

export function trackPageView(path: string): void {
  if (!initialized || !activeMeasurementId || !window.gtag) return

  const normalizedPath = normalizeAnalyticsPagePath(path)
  if (!normalizedPath) return

  const pageLocation = new URL(normalizedPath, window.location.origin)

  window.gtag('event', 'page_view', {
    page_path: normalizedPath,
    page_location: pageLocation.href,
    page_referrer:
      typeof document.referrer === 'string' && document.referrer
        ? sanitizeAttributionURL(document.referrer) || ''
        : '',
    hostname: pageLocation.hostname,
    page_title: document.title,
  })
}

export function normalizeAnalyticsPagePath(path: string): string | null {
  const rawPath = String(path ?? '').trim()
  if (rawPath === '') return null

  const lowerPath = rawPath.toLowerCase()
  if (
    lowerPath === 'undefined' ||
    lowerPath === '/undefined' ||
    lowerPath === 'null' ||
    lowerPath === '/null'
  ) {
    return null
  }

  const candidate =
    rawPath.startsWith('/') || /^[a-z][a-z0-9+.-]*:\/\//i.test(rawPath)
      ? rawPath
      : `/${rawPath}`

  try {
    const url = new URL(candidate, window.location.origin)
    const normalizedSearch = normalizeSearchParams(url.search)
    return `${url.pathname}${normalizedSearch}`
  } catch {
    return null
  }
}

function normalizeSearchParams(search: string): string {
  if (!search) return ''
  const raw = search.slice(1)
  const duplicateSeparator = raw.indexOf('?')
  if (
    duplicateSeparator > 0 &&
    raw.slice(0, duplicateSeparator) === raw.slice(duplicateSeparator + 1)
  ) {
    return `?${raw.slice(0, duplicateSeparator)}`
  }
  return search
}

export function trackAnalyticsEvent(
  eventName: string,
  params: Record<string, unknown> = {}
): void {
  if (!initialized || !activeMeasurementId || !window.gtag) return
  if (eventName.trim() === '') return

  window.gtag('event', eventName, params)
}

export function resetAnalyticsForTests(): void {
  activeMeasurementId = ''
  initialized = false
}
