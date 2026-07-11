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
import { getGoogleAnalyticsSessionCookieName } from './google-analytics-config'

const STORAGE_KEY = 'lizh_first_touch_attribution'

const CLICK_ID_PARAMS = ['gclid', 'fbclid', 'ttclid', 'yclid']
const UTM_PARAMS = [
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_term',
  'utm_content',
]
const SAFE_URL_PARAMS = [...UTM_PARAMS, ...CLICK_ID_PARAMS, 'aff']
const ATTRIBUTION_FIELDS = [
  'client_id',
  'session_id',
  'page_location',
  'page_referrer',
  'source',
  'medium',
  'campaign',
  'term',
  'content',
  'gclid',
  'fbclid',
  'ttclid',
  'yclid',
  'first_visit_at',
] as const

export interface FirstTouchAttribution {
  client_id?: string
  session_id?: string
  page_location?: string
  page_referrer?: string
  source?: string
  medium?: string
  campaign?: string
  term?: string
  content?: string
  gclid?: string
  fbclid?: string
  ttclid?: string
  yclid?: string
  first_visit_at?: string
}

function readGAClientID(): string {
  if (typeof document === 'undefined') return ''
  const match = document.cookie.match(/(?:^|;\s*)_ga=([^;]+)/)
  if (!match) return ''
  const parts = safeDecodeURIComponent(match[1]).split('.')
  if (parts.length < 4) return ''
  const first = parts[parts.length - 2]
  const second = parts[parts.length - 1]
  if (!/^\d+$/.test(first) || !/^\d+$/.test(second)) return ''
  return `${first}.${second}`
}

function readGASessionID(): string {
  if (typeof document === 'undefined') return ''
  const sessionCookieName = getGoogleAnalyticsSessionCookieName()
  if (!sessionCookieName) return ''
  for (const item of document.cookie.split(';')) {
    const cookie = item.trim()
    const separator = cookie.indexOf('=')
    if (separator <= 0 || cookie.slice(0, separator) !== sessionCookieName) {
      continue
    }
    const value = safeDecodeURIComponent(cookie.slice(separator + 1))
    const gs2Session = value.match(/(?:^|\$)s(\d+)(?:\$|$)/)?.[1]
    if (gs2Session) return gs2Session

    const parts = value.split('.')
    if (/^GS\d+$/.test(parts[0]) && /^\d+$/.test(parts[2] || '')) {
      return parts[2]
    }
  }
  return ''
}

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value)
  } catch {
    return ''
  }
}

function readStoredAttribution(): FirstTouchAttribution | null | undefined {
  if (typeof localStorage === 'undefined') return undefined
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === null) return undefined
    const parsed = JSON.parse(raw)
    if (
      parsed === null ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed) ||
      Object.getPrototypeOf(parsed) !== Object.prototype
    ) {
      return null
    }

    const attribution: FirstTouchAttribution = {}
    let hasAttributionField = false
    for (const field of ATTRIBUTION_FIELDS) {
      if (!Object.prototype.hasOwnProperty.call(parsed, field)) continue
      hasAttributionField = true
      if (typeof parsed[field] !== 'string') return null
      attribution[field] = parsed[field]
    }
    return hasAttributionField ? attribution : null
  } catch {
    return null
  }
}

function writeStoredAttribution(attribution: FirstTouchAttribution): void {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(attribution))
  } catch {
    // Attribution must never block the registration flow.
  }
}

export function sanitizeAttributionURL(raw: string): string | undefined {
  try {
    const url = new URL(raw)
    const safeParams = new URLSearchParams()
    for (const key of SAFE_URL_PARAMS) {
      const values = url.searchParams.getAll(key)
      for (const value of values) {
        const trimmed = value.trim()
        if (trimmed) safeParams.append(key, trimmed)
      }
    }
    const query = safeParams.toString()
    return `${url.origin}${url.pathname}${query ? `?${query}` : ''}`
  } catch {
    return undefined
  }
}

export function initializeFirstTouchAttribution(): void {
  if (typeof window === 'undefined') return

  const existing = readStoredAttribution()
  const clientID = readGAClientID()
  const sessionID = readGASessionID()
  if (existing === null) return
  if (existing) {
    const refreshed = { ...existing }
    let changed = false
    if (!refreshed.client_id && clientID) {
      refreshed.client_id = clientID
      changed = true
    }
    if (sessionID && refreshed.session_id !== sessionID) {
      refreshed.session_id = sessionID
      changed = true
    }
    if (changed) {
      writeStoredAttribution(refreshed)
    }
    return
  }

  const params = new URLSearchParams(window.location.search)
  const attribution: FirstTouchAttribution = {
    page_location: sanitizeAttributionURL(window.location.href),
    page_referrer: document.referrer
      ? sanitizeAttributionURL(document.referrer)
      : undefined,
    first_visit_at: new Date().toISOString(),
  }
  if (clientID) attribution.client_id = clientID
  if (sessionID) attribution.session_id = sessionID

  const utmMap: Record<string, keyof FirstTouchAttribution> = {
    utm_source: 'source',
    utm_medium: 'medium',
    utm_campaign: 'campaign',
    utm_term: 'term',
    utm_content: 'content',
  }
  for (const key of UTM_PARAMS) {
    const value = params.get(key)?.trim()
    if (value) attribution[utmMap[key]] = value
  }
  for (const key of CLICK_ID_PARAMS) {
    const value = params.get(key)?.trim()
    if (value) attribution[key as keyof FirstTouchAttribution] = value
  }

  writeStoredAttribution(attribution)
}

export function getFirstTouchAttribution(): FirstTouchAttribution | undefined {
  initializeFirstTouchAttribution()
  const attribution = readStoredAttribution()
  return attribution || undefined
}

export function withFirstTouchAttribution<T extends object>(
  request: T
): T & { attribution?: FirstTouchAttribution } {
  const attribution = getFirstTouchAttribution()
  return attribution ? { ...request, attribution } : request
}
