import { beforeEach, describe, expect, mock, test } from 'bun:test'

const post = mock(async () => ({ data: { success: true } }))
const put = mock(async () => ({ data: { success: true } }))
const withFirstTouchAttribution = mock(<T extends object>(request: T) => ({
  ...request,
  attribution: {
    client_id: '123.456',
    session_id: '789',
    source: 'google',
    gclid: 'click',
  },
}))

mock.module('@/lib/api', () => ({
  api: { post, put },
}))

mock.module('@/lib/first-touch-attribution', () => ({
  withFirstTouchAttribution,
}))

const {
  batchDeleteApiKeys,
  createApiKey,
  fetchTokenKeysBatch,
  updateApiKey,
  updateApiKeyStatus,
} = await import('../src/features/keys/api')

const formData = {
  name: 'tracked-key',
  remain_quota: 100,
  expired_time: -1,
  unlimited_quota: true,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
  group: 'default',
  cross_group_retry: false,
}

beforeEach(() => {
  post.mockClear()
  put.mockClear()
  withFirstTouchAttribution.mockClear()
})

describe('API key attribution', () => {
  test('createApiKey sends first-touch attribution', async () => {
    await createApiKey(formData)

    expect(withFirstTouchAttribution).toHaveBeenCalledWith(formData)
    expect(post).toHaveBeenCalledWith('/api/token/', {
      ...formData,
      attribution: {
        client_id: '123.456',
        session_id: '789',
        source: 'google',
        gclid: 'click',
      },
    })
  })

  test('updateApiKey does not send first-touch attribution', async () => {
    await updateApiKey({ ...formData, id: 7 })

    expect(withFirstTouchAttribution).not.toHaveBeenCalled()
    expect(put).toHaveBeenCalledWith('/api/token/', { ...formData, id: 7 })
  })

  test('batchDeleteApiKeys sends only token ids', async () => {
    await batchDeleteApiKeys([7, 8])

    expect(withFirstTouchAttribution).not.toHaveBeenCalled()
    expect(post).toHaveBeenCalledWith('/api/token/batch', { ids: [7, 8] })
  })

  test('updateApiKeyStatus sends only id and status', async () => {
    await updateApiKeyStatus(7, 2)

    expect(withFirstTouchAttribution).not.toHaveBeenCalled()
    expect(put).toHaveBeenCalledWith('/api/token/?status_only=true', {
      id: 7,
      status: 2,
    })
  })

  test('fetchTokenKeysBatch sends only token ids', async () => {
    await fetchTokenKeysBatch([7, 8])

    expect(withFirstTouchAttribution).not.toHaveBeenCalled()
    expect(post).toHaveBeenCalledWith('/api/token/batch/keys', { ids: [7, 8] })
  })
})
