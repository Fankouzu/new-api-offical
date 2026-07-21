import { useContext } from 'react'
import {
  missingSkinPortalProvider,
  SkinPortalContext,
} from './skin-portal-context'

export function useSkinPortalContainer(): HTMLElement | null {
  const container = useContext(SkinPortalContext)

  if (container === missingSkinPortalProvider) {
    throw new Error('useSkinPortalContainer must be used within a SkinBoundary')
  }

  return container
}
