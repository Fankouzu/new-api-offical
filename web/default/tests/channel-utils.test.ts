import { describe, expect, test } from 'bun:test'

import {
  getChannelTypeIcon,
  getChannelTypeLabel,
} from '../src/features/channels/lib/channel-utils'

describe('channel type mappings', () => {
  test('maps PingXingShiJie channel type to its label and favicon URL', () => {
    expect(getChannelTypeLabel(58)).toBe('PingXingShiJie')
    expect(getChannelTypeIcon(58)).toBe(
      'https://www.pingxingshijie.cn/favicon.ico'
    )
  })

  test('maps KieAI channel type to its label and logo URL', () => {
    expect(getChannelTypeLabel(59)).toBe('KieAI')
    expect(getChannelTypeIcon(59)).toBe('https://kie.ai/logo.png')
  })
})
