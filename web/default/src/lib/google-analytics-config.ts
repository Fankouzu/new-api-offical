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
declare global {
  interface Window {
    __GOOGLE_ANALYTICS_ID__?: string
  }
}

export function getGoogleAnalyticsMeasurementId(): string {
  const runtimeMeasurementId =
    typeof window !== 'undefined' ? window.__GOOGLE_ANALYTICS_ID__ : ''
  return (
    runtimeMeasurementId ||
    import.meta.env?.VITE_GOOGLE_ANALYTICS_ID ||
    ''
  ).trim()
}

export function getGoogleAnalyticsSessionCookieName(): string {
  const measurementId = getGoogleAnalyticsMeasurementId()
  const match = /^G-([a-z0-9]+)$/i.exec(measurementId)
  return match ? `_ga_${match[1]}` : ''
}
