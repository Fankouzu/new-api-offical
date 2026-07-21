import { createContext } from 'react'

export const missingSkinPortalProvider = Symbol('missing-skin-portal-provider')

export const SkinPortalContext = createContext<
  HTMLElement | null | typeof missingSkinPortalProvider
>(missingSkinPortalProvider)
