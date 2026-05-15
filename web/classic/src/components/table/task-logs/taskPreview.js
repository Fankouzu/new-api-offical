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

export function extractImageUrlFromTaskData(data) {
  if (data == null) return '';
  const walk = (v) => {
    if (v == null) return '';
    if (typeof v === 'string') {
      const value = v.trim();
      return /^data:image\/[^;,]+;base64,/i.test(value) ? value : '';
    }
    if (typeof v !== 'object') return '';
    if (typeof v.b64_json === 'string' && v.b64_json.trim()) {
      return `data:image/png;base64,${v.b64_json.trim()}`;
    }
    if (
      typeof v.url === 'string' &&
      /^data:image\/[^;,]+;base64,/i.test(v.url.trim())
    ) {
      return v.url.trim();
    }
    if (typeof v.url === 'string' && /^https?:\/\//.test(v.url)) {
      const lower = v.url.toLowerCase();
      if (
        /\.(jpe?g|png|webp|gif)(\?|$)/i.test(v.url) ||
        lower.includes('seedream') ||
        (lower.includes('tos-') && lower.includes('jpeg'))
      ) {
        return v.url;
      }
    }
    if (Array.isArray(v)) {
      for (const item of v) {
        const found = walk(item);
        if (found) return found;
      }
      return '';
    }
    for (const k of Object.keys(v)) {
      const found = walk(v[k]);
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
