import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { pricingSearchSchema } from './search'

describe('pricingSearchSchema', () => {
  it('accepts compare-route pricing search values', () => {
    assert.deepEqual(
      pricingSearchSchema.parse({ search: 'x', view: 'table' }),
      { search: 'x', view: 'table' }
    )
  })

  it('normalizes an unsupported view to the existing default behavior', () => {
    assert.deepEqual(pricingSearchSchema.parse({ view: 'grid' }), {
      view: undefined,
    })
  })
})
