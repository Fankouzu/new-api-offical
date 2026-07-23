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
import assert from 'node:assert/strict'
import test from 'node:test'
import { shouldSuppressNetworkToast } from './api-error-policy'

test('suppresses background-style GET failures without an HTTP response', () => {
  assert.equal(shouldSuppressNetworkToast({ config: { method: 'get' } }), true)
})

test('keeps network failure toasts for write requests', () => {
  assert.equal(
    shouldSuppressNetworkToast({ config: { method: 'post' } }),
    false
  )
})

test('keeps server response errors visible for GET requests', () => {
  assert.equal(
    shouldSuppressNetworkToast({
      config: { method: 'get' },
      response: { status: 503 },
    }),
    false
  )
})
