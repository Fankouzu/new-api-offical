const STORAGE_KEY = 'lizh_first_touch_attribution'

const CLICK_ID_PARAMS = ['gclid', 'fbclid', 'ttclid', 'yclid']
const UTM_PARAMS = [
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_term',
  'utm_content',
]

export interface FirstTouchAttribution {
  client_id?: string
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
  const parts = decodeURIComponent(match[1]).split('.')
  if (parts.length < 4) return ''
  const first = parts[parts.length - 2]
  const second = parts[parts.length - 1]
  if (!/^\d+$/.test(first) || !/^\d+$/.test(second)) return ''
  return `${first}.${second}`
}

function readStoredAttribution(): FirstTouchAttribution | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object'
      ? (parsed as FirstTouchAttribution)
      : null
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

export function initializeFirstTouchAttribution(): void {
  if (typeof window === 'undefined') return

  const existing = readStoredAttribution()
  const clientID = readGAClientID()
  if (existing) {
    if (!existing.client_id && clientID) {
      writeStoredAttribution({ ...existing, client_id: clientID })
    }
    return
  }

  const params = new URLSearchParams(window.location.search)
  const attribution: FirstTouchAttribution = {
    page_location: window.location.href,
    page_referrer: document.referrer || undefined,
    first_visit_at: new Date().toISOString(),
  }
  if (clientID) attribution.client_id = clientID

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
