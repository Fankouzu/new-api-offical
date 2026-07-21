import { createContext, createElement, useContext, type ReactNode } from 'react'

const missingSkinPortalProvider = Symbol('missing-skin-portal-provider')

const SkinPortalContext = createContext<
  HTMLElement | null | typeof missingSkinPortalProvider
>(missingSkinPortalProvider)

type SkinPortalProviderProps = {
  children: ReactNode
  container: HTMLElement | null
}

export function SkinPortalProvider(props: SkinPortalProviderProps) {
  return createElement(
    SkinPortalContext.Provider,
    { value: props.container },
    props.children
  )
}

export function useSkinPortalContainer(): HTMLElement | null {
  const container = useContext(SkinPortalContext)

  if (container === missingSkinPortalProvider) {
    throw new Error('useSkinPortalContainer must be used within a SkinBoundary')
  }

  return container
}
