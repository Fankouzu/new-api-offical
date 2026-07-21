import type { ReactNode } from 'react'
import '../styles/index.css'

type CustomPublicShellProps = {
  children: ReactNode
}

export function CustomPublicShell(props: CustomPublicShellProps) {
  return <div data-skin-shell='custom'>{props.children}</div>
}
