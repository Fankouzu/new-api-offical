import {
  DEFAULT_TOKEN_UNIT,
  ENDPOINT_TYPES,
  FILTER_ALL,
  QUOTA_TYPES,
  SORT_OPTIONS,
  VIEW_MODES,
  type ViewMode,
} from '../constants'
import type { PricingSearch } from './search'

export type FilterState = PricingSearch

export function createInitialFilterState(search: PricingSearch): FilterState {
  return { ...search }
}

function normalizeViewMode(value: unknown): ViewMode {
  return value === VIEW_MODES.TABLE ? VIEW_MODES.TABLE : VIEW_MODES.CARD
}

export function getFilterDefaults(filterState: FilterState) {
  return {
    searchInput: filterState.search || '',
    sortBy: filterState.sort || SORT_OPTIONS.NAME,
    vendorFilter: filterState.vendor || FILTER_ALL,
    groupFilter: filterState.group || FILTER_ALL,
    quotaTypeFilter: filterState.quotaType || QUOTA_TYPES.ALL,
    endpointTypeFilter: filterState.endpointType || ENDPOINT_TYPES.ALL,
    tagFilter: filterState.tag || FILTER_ALL,
    tokenUnit: filterState.tokenUnit === 'K' ? 'K' : DEFAULT_TOKEN_UNIT,
    viewMode: normalizeViewMode(filterState.view),
    showRechargePrice: filterState.rechargePrice === true,
  }
}
