type SkinSurfaceDataset = {
  skinSurface?: string
}

export type SkinSurfaceToken = symbol

const registrationsByDataset = new WeakMap<
  SkinSurfaceDataset,
  Map<SkinSurfaceToken, string>
>()

export function createSkinSurfaceToken(): SkinSurfaceToken {
  return Symbol('skin-surface-owner')
}

function syncActiveSkinSurface(
  dataset: SkinSurfaceDataset,
  registrations: Map<SkinSurfaceToken, string>
): void {
  const activeSkinId = Array.from(registrations.values()).at(-1)

  if (activeSkinId === undefined) {
    delete dataset.skinSurface
    registrationsByDataset.delete(dataset)
    return
  }

  dataset.skinSurface = activeSkinId
}

export function registerSkinSurface(
  dataset: SkinSurfaceDataset,
  token: SkinSurfaceToken,
  skinId: string
): void {
  const registrations = registrationsByDataset.get(dataset) ?? new Map()
  const currentSkinId = registrations.get(token)

  if (!registrationsByDataset.has(dataset)) {
    registrationsByDataset.set(dataset, registrations)
  }

  if (currentSkinId !== skinId) {
    registrations.delete(token)
    registrations.set(token, skinId)
  }

  syncActiveSkinSurface(dataset, registrations)
}

export function unregisterSkinSurface(
  dataset: SkinSurfaceDataset,
  token: SkinSurfaceToken
): void {
  const registrations = registrationsByDataset.get(dataset)

  if (registrations === undefined) {
    return
  }

  registrations.delete(token)
  syncActiveSkinSurface(dataset, registrations)
}
