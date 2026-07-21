import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { createInitialFilterState, getFilterDefaults } from './filter-state'
import { pricingSearchSchema } from './search'

describe('pricing filter initialization', () => {
  it('initializes compare-route search without pricing route ownership', () => {
    const search = pricingSearchSchema.parse({ search: 'x', view: 'table' })

    assert.deepEqual(createInitialFilterState(search), {
      search: 'x',
      view: 'table',
    })
  })

  it('preserves existing defaults for absent search values', () => {
    assert.deepEqual(getFilterDefaults(createInitialFilterState({})), {
      searchInput: '',
      sortBy: 'name',
      vendorFilter: 'all',
      groupFilter: 'all',
      quotaTypeFilter: 'all',
      endpointTypeFilter: 'all',
      tagFilter: 'all',
      tokenUnit: 'M',
      viewMode: 'card',
      showRechargePrice: false,
    })
  })
})
