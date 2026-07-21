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

import { inferModelMetadata } from '../features/pricing/lib/model-metadata'
import type { PricingModel } from '../features/pricing/types'

function pricingModel(overrides: Partial<PricingModel> = {}): PricingModel {
  return {
    id: 1,
    model_name: 'catalog-model-without-reported-capabilities',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    supported_endpoint_types: ['openai'],
    ...overrides,
  }
}

test('does not invent capabilities when the catalog reports none', () => {
  const metadata = inferModelMetadata(pricingModel())

  assert.deepEqual(metadata.capabilities, [])
})

test('preserves explicitly reported catalog capabilities', () => {
  const metadata = inferModelMetadata(
    pricingModel({ capabilities: ['reasoning', 'tools'] })
  )

  assert.deepEqual(metadata.capabilities, ['reasoning', 'tools'])
})

test('does not infer modalities from a model name', () => {
  const metadata = inferModelMetadata(
    pricingModel({ model_name: 'vision-audio-video-marketing-name' })
  )

  assert.deepEqual(metadata.input_modalities, ['text'])
  assert.deepEqual(metadata.output_modalities, ['text'])
})
