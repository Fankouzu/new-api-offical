import { describe, expect, test } from 'bun:test'

import { extractTaskMediaResults } from '../src/features/usage-logs/lib/task-media-results'

describe('extractTaskMediaResults', () => {
  test('extracts multiple image URLs from task data and result URL', () => {
    const results = extractTaskMediaResults({
      action: 'GENERATE',
      data: JSON.stringify({
        data: [
          { url: 'https://example.com/first.png' },
          { image_url: 'https://example.com/second.webp' },
        ],
      }),
      result_url: 'https://example.com/cover.jpg',
      status: 'SUCCESS',
      task_id: 'task-image',
      upstream_kind: 'image',
    })

    expect(results).toEqual([
      { type: 'image', url: 'https://example.com/cover.jpg' },
      { type: 'image', url: 'https://example.com/first.png' },
      { type: 'image', url: 'https://example.com/second.webp' },
    ])
  })

  test('prefers base64 image data from task data over task result proxy URL', () => {
    const results = extractTaskMediaResults({
      action: 'generate',
      data: JSON.stringify({
        created: 1778776346,
        data: [{ url: 'data:image/png;base64,abc123' }],
      }),
      result_url: '/api/task/642/result',
      status: 'SUCCESS',
      task_id: 'task_PjvoEZY3moE8gh7MXZxs6HoCUr0GUc0s',
      upstream_kind: 'image',
    })

    expect(results).toEqual([{ type: 'image', url: 'data:image/png;base64,abc123' }])
  })

  test('extracts multiple video URLs from nested task payloads', () => {
    const results = extractTaskMediaResults({
      action: 'TEXT_GENERATE',
      data: {
        content: { video_url: 'https://example.com/generated.mp4' },
        videos: [{ url: 'https://cdn.example.com/alt.webm' }],
      },
      result_url: 'https://example.com/generated.mp4',
      status: 'SUCCESS',
      task_id: 'task-video',
      upstream_kind: 'video',
    })

    expect(results).toEqual([
      { type: 'video', url: 'https://example.com/generated.mp4' },
      { type: 'video', url: 'https://cdn.example.com/alt.webm' },
    ])
  })

  test('uses legacy fail reason URL when result URL is absent', () => {
    const results = extractTaskMediaResults({
      action: 'GENERATE',
      fail_reason: 'https://legacy.example.com/result.mp4',
      status: 'SUCCESS',
      task_id: 'task-legacy',
    })

    expect(results).toEqual([
      { type: 'video', url: 'https://legacy.example.com/result.mp4' },
    ])
  })

  test('ignores stale video proxy URL for image tasks and avoids input-only URLs', () => {
    const results = extractTaskMediaResults({
      action: 'GENERATE',
      data: {
        request: {
          input_image: 'https://uploads.example.com/user-input.png',
        },
        data: [{ url: 'https://example.com/generated-seedream.jpeg' }],
      },
      result_url: 'https://api.example.com/v1/videos/task-image/content',
      status: 'SUCCESS',
      task_id: 'task-image',
      upstream_kind: 'image',
    })

    expect(results).toEqual([
      { type: 'image', url: 'https://example.com/generated-seedream.jpeg' },
    ])
  })

  test('ignores legacy video proxy fail reason when image result already exists', () => {
    const results = extractTaskMediaResults({
      action: 'GENERATE',
      fail_reason: 'https://api.example.com/v1/videos/task-legacy-image/content',
      result_url: 'https://example.com/generated-image.png',
      status: 'SUCCESS',
      task_id: 'task-legacy-image',
    })

    expect(results).toEqual([
      { type: 'image', url: 'https://example.com/generated-image.png' },
    ])
  })
})
