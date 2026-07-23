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
import { AxiosError } from 'axios'
import assert from 'node:assert/strict'
import test from 'node:test'
import { shouldRetryQuery } from './query-retry-policy'

test('retries a response-less network error only once in production', () => {
  const error = new AxiosError('Network Error')

  assert.equal(shouldRetryQuery(0, error, true), true)
  assert.equal(shouldRetryQuery(1, error, true), false)
})

test('does not retry authorization failures', () => {
  const error = new AxiosError('Forbidden', undefined, undefined, undefined, {
    status: 403,
  } as never)

  assert.equal(shouldRetryQuery(0, error, true), false)
})

test('keeps the existing production retry budget for other failures', () => {
  const error = new Error('temporary query failure')

  assert.equal(shouldRetryQuery(2, error, true), true)
  assert.equal(shouldRetryQuery(3, error, true), false)
})

test('does not retry queries during development', () => {
  assert.equal(shouldRetryQuery(0, new Error('failure'), false), false)
})
