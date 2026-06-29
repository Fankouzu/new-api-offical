/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { type ComponentType, type SVGProps } from 'react'
import { Link } from '@tanstack/react-router'
import { IconTelegram } from '@/assets/brand-icons'
import { cn } from '@/lib/utils'
import type { TopNavLink } from '../types'

type NavLinkIcon = ComponentType<SVGProps<SVGSVGElement>>

const TOP_NAV_LINK_ICONS: Record<string, NavLinkIcon> = {
  telegram: IconTelegram,
}

export function getTopNavLinkIcon(link: TopNavLink): NavLinkIcon | null {
  if (!link.icon) return null

  return TOP_NAV_LINK_ICONS[link.icon.trim().toLowerCase()] ?? null
}

export function shouldRenderTopNavLinkAsIcon(link: TopNavLink): boolean {
  return link.display === 'icon' && getTopNavLinkIcon(link) !== null
}

export function renderTopNavLinkContent(
  link: TopNavLink,
  label: string,
  options: { iconOnly?: boolean; showIconWithText?: boolean } = {}
) {
  const Icon = getTopNavLinkIcon(link)

  if (options.iconOnly && Icon) {
    return (
      <>
        <Icon aria-hidden='true' />
        <span className='sr-only'>{label}</span>
      </>
    )
  }

  if (options.showIconWithText && Icon) {
    return (
      <>
        <Icon aria-hidden='true' />
        <span>{label}</span>
      </>
    )
  }

  return label
}

export function TopNavLinkIconList({
  links,
  className,
  getLabel = (link) => link.title,
}: {
  links: TopNavLink[]
  className?: string
  getLabel?: (link: TopNavLink) => string
}) {
  if (links.length === 0) return null

  return (
    <>
      {links.map((link) => {
        const label = getLabel(link)
        const iconOnly = shouldRenderTopNavLinkAsIcon(link)
        const linkClassName = cn(
          'text-muted-foreground hover:text-foreground inline-flex size-9 items-center justify-center rounded-md transition-colors',
          link.disabled && 'pointer-events-none opacity-50',
          className
        )

        if (link.external) {
          return (
            <a
              key={`${link.title}-${link.href}`}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              className={linkClassName}
              aria-label={label}
              aria-disabled={link.disabled}
            >
              {renderTopNavLinkContent(link, label, { iconOnly })}
            </a>
          )
        }

        return (
          <Link
            key={`${link.title}-${link.href}`}
            to={link.href}
            className={linkClassName}
            disabled={link.disabled}
            aria-label={label}
          >
            {renderTopNavLinkContent(link, label, { iconOnly })}
          </Link>
        )
      })}
    </>
  )
}
