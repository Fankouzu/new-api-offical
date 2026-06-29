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
import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import type { TopNavLink } from '../types'
import {
  getExternalTopNavLinkProps,
  renderTopNavLinkContent,
  shouldRenderTopNavLinkAsIcon,
} from './top-nav-link-utils'

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
          'text-muted-foreground hover:text-foreground inline-flex items-center justify-center rounded-md transition-colors',
          iconOnly
            ? 'size-9'
            : 'h-9 whitespace-nowrap px-3 text-sm font-medium',
          link.disabled && 'pointer-events-none opacity-50',
          className
        )

        if (link.external) {
          return (
            <a
              key={`${link.title}-${link.href}`}
              {...getExternalTopNavLinkProps(link)}
              className={linkClassName}
              aria-label={iconOnly ? label : undefined}
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
            aria-label={iconOnly ? label : undefined}
          >
            {renderTopNavLinkContent(link, label, { iconOnly })}
          </Link>
        )
      })}
    </>
  )
}
