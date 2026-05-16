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

const IMAGE_DATA_URL_PATTERN = /^data:image\/[^;,]+;base64,/i;
const VIDEO_DATA_URL_PATTERN = /^data:video\/[^;,]+;base64,/i;
const IMAGE_URL_PATTERN = /\.(jpe?g|png|webp|gif|bmp|avif)(\?|#|$)/i;
const VIDEO_URL_PATTERN = /\.(mp4|webm|mov|m4v|avi|mkv|m3u8)(\?|#|$)/i;
const MEDIA_KEY_PATTERN = /url|image|img|video|thumbnail|cover/i;
const INPUT_KEY_PATTERN = /request|input|prompt|source|reference|mask/i;

function looksLikeImageUrl(value) {
  const lower = value.toLowerCase();
  return (
    IMAGE_URL_PATTERN.test(value) ||
    lower.includes('seedream') ||
    (lower.includes('tos-') && lower.includes('jpeg'))
  );
}

function looksLikeVideoUrl(value) {
  const lower = value.toLowerCase();
  return VIDEO_URL_PATTERN.test(value) || lower.includes('video');
}

function classifyMediaUrl(value, keyHint = '') {
  if (typeof value !== 'string') return null;
  const url = value.trim();
  if (!url) return null;
  if (IMAGE_DATA_URL_PATTERN.test(url)) return { type: 'image', url };
  if (VIDEO_DATA_URL_PATTERN.test(url)) return { type: 'video', url };
  if (!/^https?:\/\//i.test(url)) return null;

  if (looksLikeImageUrl(url)) return { type: 'image', url };
  if (looksLikeVideoUrl(url)) return { type: 'video', url };

  const normalizedKey = String(keyHint || '');
  if (!INPUT_KEY_PATTERN.test(normalizedKey) && MEDIA_KEY_PATTERN.test(normalizedKey)) {
    if (/video/i.test(normalizedKey)) return { type: 'video', url };
    if (/image|img|thumbnail|cover/i.test(normalizedKey)) return { type: 'image', url };
  }
  return null;
}

export function extractMediaResultFromTaskData(data) {
  if (data == null) return '';
  const walk = (v, keyHint = '') => {
    if (v == null) return '';
    if (typeof v === 'string') {
      if (INPUT_KEY_PATTERN.test(keyHint)) return '';
      return classifyMediaUrl(v, keyHint) || '';
    }
    if (typeof v !== 'object') return '';
    if (typeof v.b64_json === 'string' && v.b64_json.trim()) {
      return { type: 'image', url: `data:image/png;base64,${v.b64_json.trim()}` };
    }
    const directUrl = classifyMediaUrl(v.url, 'url');
    if (directUrl) {
      return directUrl;
    }
    if (Array.isArray(v)) {
      for (const item of v) {
        const found = walk(item, keyHint);
        if (found) return found;
      }
      return '';
    }
    for (const k of Object.keys(v)) {
      const found = walk(v[k], k);
      if (found) return found;
    }
    return '';
  };
  try {
    const obj = typeof data === 'string' ? JSON.parse(data) : data;
    return walk(obj);
  } catch {
    return '';
  }
}

export function extractImageUrlFromTaskData(data) {
  const result = extractMediaResultFromTaskData(data);
  return result?.type === 'image' ? result.url : '';
}

export function resolveTaskPreviewUrl(record, options = {}) {
  const { allowDataFallback = false } = options;
  const primary = record?.result_url;
  if (
    typeof primary !== 'string' ||
    (!/^https?:\/\//.test(primary) &&
      !/^data:image\/[^;,]+;base64,/i.test(primary.trim()))
  ) {
    return allowDataFallback
      ? extractImageUrlFromTaskData(record?.data) || ''
      : '';
  }
  if (
    record.upstream_kind === 'image' &&
    primary.includes('/v1/videos/') &&
    primary.includes('/content')
  ) {
    const fromData = allowDataFallback
      ? extractImageUrlFromTaskData(record.data)
      : '';
    if (fromData) return fromData;
  }
  return primary;
}
