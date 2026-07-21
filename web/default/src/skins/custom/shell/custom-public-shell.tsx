import { createElement, type ReactNode } from 'react'
import '../styles/index.css'

type CustomPublicShellProps = {
  children: ReactNode
}

export function CustomPublicShell(props: CustomPublicShellProps) {
  return createElement('div', { 'data-skin-shell': 'custom' }, props.children)
}
