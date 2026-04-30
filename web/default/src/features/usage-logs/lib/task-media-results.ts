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
const IMAGE_URL_PATTERN = /\.(jpe?g|png|webp|gif|bmp|avif)(\?|#|$)/i
const VIDEO_URL_PATTERN = /\.(mp4|webm|mov|m4v|avi|mkv|m3u8)(\?|#|$)/i
const VIDEO_ACTIONS = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])
const RESULT_KEY_PATTERN = /(result|output|generated|media|asset|content)/i
const IMAGE_KEY_PATTERN = /image|img|thumbnail|cover|first_frame|last_frame/i
const VIDEO_KEY_PATTERN = /video/i
const INPUT_KEY_PATTERN = /(request|input|prompt|source|reference|mask)/i

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function parseTaskData(data: unknown): unknown {
  if (typeof data !== 'string') return data
  const trimmed = data.trim()
  if (!trimmed) return undefined

  try {
    return JSON.parse(trimmed) as unknown
  } catch {
    return undefined
  }
}

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && HTTP_URL_PATTERN.test(value.trim())
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
  if (!isHttpUrl(urlValue)) return

  const url = urlValue.trim()
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

function walkTaskData(
  value: unknown,
  source: TaskMediaSource,
  results: TaskMediaResult[],
  seen: Set<string>,
  keyHint?: string
): void {
  if (Array.isArray(value)) {
    for (const item of value) {
      walkTaskData(item, source, results, seen, keyHint)
    }
    return
  }

  if (!isRecord(value)) {
    addMediaResult(
      results,
      seen,
      source,
      value,
      keyHint,
      RESULT_KEY_PATTERN.test(keyHint ?? '')
    )
    return
  }

  for (const [key, nestedValue] of Object.entries(value)) {
    const nestedKeyHint = keyHint ? `${keyHint}.${key}` : key
    const allowTaskFallback = RESULT_KEY_PATTERN.test(nestedKeyHint)
    if (isHttpUrl(nestedValue)) {
      addMediaResult(
        results,
        seen,
        source,
        nestedValue,
        nestedKeyHint,
        allowTaskFallback
      )
      continue
    }

    walkTaskData(nestedValue, source, results, seen, nestedKeyHint)
  }
}

export function extractTaskMediaResults(source: TaskMediaSource): TaskMediaResult[] {
  if (source.status !== TASK_STATUS.SUCCESS) return []

  const results: TaskMediaResult[] = []
  const seen = new Set<string>()

  addMediaResult(results, seen, source, source.result_url, 'result_url', true)
  addMediaResult(results, seen, source, source.fail_reason, 'fail_reason', true)
  walkTaskData(parseTaskData(source.data), source, results, seen)

  return results
}
