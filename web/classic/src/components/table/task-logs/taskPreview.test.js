import { describe, expect, test } from 'bun:test';
import {
  extractImageUrlFromTaskData,
  resolveTaskPreviewUrl,
} from './taskPreview.js';

describe('task preview helpers', () => {
  test('extracts image data URL from task 642-style provider payload', () => {
    const url = extractImageUrlFromTaskData(
      JSON.stringify({
        created: 1778776346,
        data: [{ url: 'data:image/png;base64,abc123' }],
      })
    );

    expect(url).toBe('data:image/png;base64,abc123');
  });

  test('can use task data fallback when list records include provider payloads', () => {
    const previewUrl = resolveTaskPreviewUrl(
      {
        data: {
          created: 1778776346,
          data: [{ url: 'data:image/png;base64,abc123' }],
        },
        result_url: '/api/task/642/result',
        upstream_kind: 'image',
      },
      { allowDataFallback: true }
    );

    expect(previewUrl).toBe('data:image/png;base64,abc123');
  });
});
