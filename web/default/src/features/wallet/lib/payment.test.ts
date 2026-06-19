import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { hasConfigurableTopup } from './payment'
import type { TopupInfo } from '../types'

function topupInfo(overrides: Partial<TopupInfo> = {}): TopupInfo {
  return {
    enable_online_topup: false,
    enable_stripe_topup: false,
    enable_creem_topup: false,
    enable_waffo_topup: false,
    enable_waffo_pancake_topup: false,
    enable_binance_pay_topup: false,
    pay_methods: [],
    min_topup: 1,
    stripe_min_topup: 1,
    waffo_min_topup: 1,
    waffo_pancake_min_topup: 1,
    binance_pay_min_topup: 1,
    amount_options: [],
    discount: {},
    topup_link: '',
    ...overrides,
  }
}

describe('wallet payment helpers', () => {
  it('treats Binance Pay as a configurable top-up method', () => {
    assert.equal(
      hasConfigurableTopup(
        topupInfo({
          enable_binance_pay_topup: true,
          pay_methods: [
            {
              name: 'Binance Pay',
              type: 'binance_pay',
              min_topup: 1,
            },
          ],
        })
      ),
      true
    )
  })

  it('returns false when no configurable top-up method is enabled', () => {
    assert.equal(hasConfigurableTopup(topupInfo()), false)
  })
})
