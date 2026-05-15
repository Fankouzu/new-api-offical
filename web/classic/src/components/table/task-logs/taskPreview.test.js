import { describe, expect, test } from 'bun:test';
import {
  extractImageUrlFromTaskData,
  resolveTaskDetailPreviewUrl,
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

  test('prefers raw task base64 over lightweight detail result proxy URL', () => {
    const previewUrl = resolveTaskDetailPreviewUrl(
      { result: { url: '/api/task/642/result', type: 'image' } },
      {
        data: {
          created: 1778776346,
          data: [{ url: 'data:image/png;base64,abc123' }],
        },
      }
    );

    expect(previewUrl).toBe('data:image/png;base64,abc123');
  });
});
