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

type GtagCommand = [command: string, ...args: unknown[]]

const DEFAULT_GOOGLE_ANALYTICS_MEASUREMENT_ID = 'G-9693VBP1VM'

declare global {
  interface Window {
    dataLayer?: GtagCommand[]
    gtag?: (...args: GtagCommand) => void
  }
}

let activeMeasurementId = ''
let initialized = false

export function getGoogleAnalyticsMeasurementId(): string {
  return (
    import.meta.env.VITE_GOOGLE_ANALYTICS_ID ||
    DEFAULT_GOOGLE_ANALYTICS_MEASUREMENT_ID
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
  window.gtag = (...args: GtagCommand) => {
    window.dataLayer?.push(args)
  }

  if (!document.querySelector('[data-google-analytics-script="true"]')) {
    const script = document.createElement('script')
    script.async = true
    script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(normalizedId)}`
    script.dataset.googleAnalyticsScript = 'true'
    document.head.appendChild(script)
  }

  window.gtag('js', new Date())
  window.gtag('config', normalizedId, { send_page_view: false })
}

export function trackPageView(path: string): void {
  if (!initialized || !activeMeasurementId || !window.gtag) return

  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const pageLocation = new URL(normalizedPath, window.location.origin).href

  window.gtag('event', 'page_view', {
    page_path: normalizedPath,
    page_location: pageLocation,
    page_title: document.title,
  })
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
