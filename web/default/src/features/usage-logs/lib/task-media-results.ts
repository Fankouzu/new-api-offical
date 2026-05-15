import { TASK_ACTIONS, TASK_STATUS } from '../constants'
import type { TaskLog } from '../types'

export type TaskMediaResult = {
  type: 'image' | 'video'
  url: string
}

type TaskMediaSource = Pick<
  TaskLog,
  | 'action'
  | 'data'
  | 'fail_reason'
  | 'result_url'
  | 'status'
  | 'task_id'
  | 'upstream_kind'
>

const HTTP_URL_PATTERN = /^https?:\/\//i
const DATA_IMAGE_URL_PATTERN = /^data:image\/[^;,]+;base64,/i
const IMAGE_URL_PATTERN = /\.(jpe?g|png|webp|gif|bmp|avif)(\?|#|$)/i
const VIDEO_URL_PATTERN = /\.(mp4|webm|mov|m4v|avi|mkv|m3u8)(\?|#|$)/i
const VIDEO_ACTIONS = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])
const IMAGE_KEY_PATTERN = /image|img|thumbnail|cover|first_frame|last_frame/i
const VIDEO_KEY_PATTERN = /video/i
const INPUT_KEY_PATTERN = /(request|input|prompt|source|reference|mask)/i

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && HTTP_URL_PATTERN.test(value.trim())
}

function isDataImageUrl(value: unknown): value is string {
  return typeof value === 'string' && DATA_IMAGE_URL_PATTERN.test(value.trim())
}

function looksLikeImageUrl(url: string): boolean {
  const lower = url.toLowerCase()
  return (
    IMAGE_URL_PATTERN.test(url) ||
    lower.includes('seedream') ||
    (lower.includes('tos-') && lower.includes('jpeg'))
  )
}

function looksLikeVideoUrl(url: string): boolean {
  return VIDEO_URL_PATTERN.test(url)
}

function isStaleImageProxyUrl(url: string, source: TaskMediaSource): boolean {
  return (
    source.upstream_kind === 'image' &&
    url.includes('/v1/videos/') &&
    url.includes('/content')
  )
}

function isTaskVideoProxyUrl(url: string, source: TaskMediaSource): boolean {
  return Boolean(
    source.task_id && url.includes(`/v1/videos/${source.task_id}/content`)
  )
}

function inferMediaType(
  url: string,
  keyHint: string | undefined,
  source: TaskMediaSource,
  allowTaskFallback: boolean
): TaskMediaResult['type'] | undefined {
  const normalizedKey = keyHint?.toLowerCase() ?? ''
  if (INPUT_KEY_PATTERN.test(normalizedKey)) return undefined

  if (IMAGE_KEY_PATTERN.test(normalizedKey) || looksLikeImageUrl(url)) {
    return 'image'
  }
  if (VIDEO_KEY_PATTERN.test(normalizedKey) || looksLikeVideoUrl(url)) {
    return 'video'
  }
  if (allowTaskFallback) {
    if (source.upstream_kind === 'image') return 'image'
    if (source.upstream_kind === 'video') return 'video'
    if (VIDEO_ACTIONS.has(source.action)) return 'video'
  }
  return undefined
}

function addMediaResult(
  results: TaskMediaResult[],
  seen: Set<string>,
  source: TaskMediaSource,
  urlValue: unknown,
  keyHint?: string,
  allowTaskFallback: boolean = false
): void {
  if (!isHttpUrl(urlValue) && !isDataImageUrl(urlValue)) return

  const url = urlValue.trim()
  if (isDataImageUrl(url)) {
    if (seen.has(url)) return
    seen.add(url)
    results.push({ type: 'image', url })
    return
  }

  if (isStaleImageProxyUrl(url, source)) return
  if (
    isTaskVideoProxyUrl(url, source) &&
    results.some((result) => result.type === 'image')
  ) {
    return
  }
  if (seen.has(url)) return

  const type = inferMediaType(url, keyHint, source, allowTaskFallback)
  if (!type) return

  seen.add(url)
  results.push({ type, url })
}

function parseTaskData(data: unknown): unknown {
  if (typeof data !== 'string') return data
  const trimmed = data.trim()
  if (!trimmed) return undefined
  try {
    return JSON.parse(trimmed) as unknown
  } catch {
    return data
  }
}

function collectMediaFromValue(
  results: TaskMediaResult[],
  seen: Set<string>,
  source: TaskMediaSource,
  value: unknown,
  keyHint?: string
): void {
  if (typeof value === 'string') {
    addMediaResult(results, seen, source, value, keyHint)
    return
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      collectMediaFromValue(results, seen, source, item, keyHint)
    }
    return
  }

  if (!value || typeof value !== 'object') return

  for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
    collectMediaFromValue(results, seen, source, child, key)
  }
}

export function extractTaskMediaResults(source: TaskMediaSource): TaskMediaResult[] {
  if (source.status !== TASK_STATUS.SUCCESS) return []

  const results: TaskMediaResult[] = []
  const seen = new Set<string>()

  addMediaResult(results, seen, source, source.result_url, 'result_url', true)
  addMediaResult(results, seen, source, source.fail_reason, 'fail_reason', true)
  collectMediaFromValue(results, seen, source, parseTaskData(source.data), 'data')

  return results
}
