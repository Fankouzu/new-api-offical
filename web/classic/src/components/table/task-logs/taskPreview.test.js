/*
Copyright (C) 2025 QuantumNous

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
