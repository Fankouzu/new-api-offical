import { createElement, type ReactNode } from 'react'
import { SkinPortalContext } from './skin-portal-context'

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
