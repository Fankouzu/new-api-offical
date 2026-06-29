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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { useStatus } from '@/hooks/use-status'
import {
  parseHeaderNavModules,
  type HeaderNavCustomLinkConfig,
  type HeaderNavCustomLinkPosition,
  type HeaderNavModulesConfig,
} from '@/features/system-settings/maintenance/config'

export type TopNavLink = {
  title: string
  href: string
  disabled?: boolean
  external?: boolean
  display?: 'text' | 'icon'
  icon?: string
}

export type TopNavLinkSlots = {
  primary: TopNavLink[]
  before_search: TopNavLink[]
  after_search: TopNavLink[]
  before_notifications: TopNavLink[]
  after_notifications: TopNavLink[]
  before_theme: TopNavLink[]
  after_theme: TopNavLink[]
  before_language: TopNavLink[]
  after_language: TopNavLink[]
}

const HEADER_NAV_UTILITY_CUSTOM_LINK_POSITIONS = [
  'before_search',
  'after_search',
  'before_notifications',
  'after_notifications',
  'before_theme',
  'after_theme',
  'before_language',
  'after_language',
] as const

/**
 * Generate top navigation links based on HeaderNavModules configuration from backend /api/status
 * Backend format example (stringified JSON):
 * {
 *   home: true,
 *   console: true,
 *   pricing: { enabled: true, requireAuth: false },
 *   rankings: { enabled: true, requireAuth: false },
 *   docs: true,
 *   about: true
 * }
 */
function customLinkToTopNavLink(
  link: HeaderNavCustomLinkConfig,
  isAuthed: boolean
): TopNavLink {
  return {
    title: link.title,
    href: link.href,
    disabled: link.requireAuth && !isAuthed,
    external: link.external,
    display: link.display,
    icon: link.icon,
  }
}

function appendCustomLinks(
  links: TopNavLink[],
  customLinks: HeaderNavCustomLinkConfig[],
  position: HeaderNavCustomLinkPosition,
  isAuthed: boolean
) {
  customLinks
    .filter((link) => link.enabled && link.position === position)
    .forEach((link) => links.push(customLinkToTopNavLink(link, isAuthed)))
}

function createEmptyTopNavLinkSlots(): TopNavLinkSlots {
  return {
    primary: [],
    before_search: [],
    after_search: [],
    before_notifications: [],
    after_notifications: [],
    before_theme: [],
    after_theme: [],
    before_language: [],
    after_language: [],
  }
}

function addUtilityCustomLinks(
  slots: TopNavLinkSlots,
  customLinks: HeaderNavCustomLinkConfig[],
  isAuthed: boolean
) {
  HEADER_NAV_UTILITY_CUSTOM_LINK_POSITIONS.forEach((position) => {
    customLinks
      .filter((link) => link.enabled && link.position === position)
      .forEach((link) =>
        slots[position].push(customLinkToTopNavLink(link, isAuthed))
      )
  })
}

export function buildTopNavLinks({
  modules,
  docsLink,
  isAuthed,
  t,
}: {
  modules: HeaderNavModulesConfig
  docsLink?: string | null
  isAuthed: boolean
  t: (key: string) => string
}): TopNavLink[] {
  const links: TopNavLink[] = []

  // Home
  if (modules?.home !== false) {
    links.push({ title: t('Home'), href: '/' })
  }

  // Console -> /dashboard (new console path)
  if (modules?.console !== false) {
    links.push({ title: t('Console'), href: '/dashboard' })
  }
  appendCustomLinks(links, modules.customLinks, 'after_console', isAuthed)

  // Pricing
  const pricing = modules?.pricing
  if (pricing && typeof pricing === 'object' && pricing.enabled) {
    const disabled = pricing.requireAuth && !isAuthed
    links.push({ title: t('Model Square'), href: '/pricing', disabled })
  }
  appendCustomLinks(links, modules.customLinks, 'after_pricing', isAuthed)

  // Rankings
  const rankings = modules?.rankings
  if (rankings && typeof rankings === 'object' && rankings.enabled) {
    const disabled = rankings.requireAuth && !isAuthed
    links.push({ title: t('Rankings'), href: '/rankings', disabled })
  }
  appendCustomLinks(links, modules.customLinks, 'after_rankings', isAuthed)

  // Docs (supports external links)
  if (modules?.docs !== false) {
    if (docsLink) {
      links.push({ title: t('Docs'), href: docsLink, external: true })
    } else {
      links.push({ title: t('Docs'), href: '/docs' })
    }
  }
  appendCustomLinks(links, modules.customLinks, 'after_docs', isAuthed)

  // About
  if (modules?.about !== false) {
    links.push({ title: t('About'), href: '/about' })
  }
  appendCustomLinks(links, modules.customLinks, 'end', isAuthed)

  return links
}

export function buildTopNavLinkSlots({
  modules,
  docsLink,
  isAuthed,
  t,
}: {
  modules: HeaderNavModulesConfig
  docsLink?: string | null
  isAuthed: boolean
  t: (key: string) => string
}): TopNavLinkSlots {
  const slots = createEmptyTopNavLinkSlots()

  slots.primary = buildTopNavLinks({
    modules,
    docsLink,
    isAuthed,
    t,
  })
  addUtilityCustomLinks(slots, modules.customLinks, isAuthed)

  return slots
}

function useHeaderNavContext() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { auth } = useAuthStore()

  const modules = useMemo(() => {
    return parseHeaderNavModules(status?.HeaderNavModules as string | undefined)
  }, [status?.HeaderNavModules])

  const docsLink: string | undefined = status?.docs_link as string | undefined
  const isAuthed = !!auth?.user

  return { modules, docsLink, isAuthed, t }
}

export function useTopNavLinks(): TopNavLink[] {
  const { modules, docsLink, isAuthed, t } = useHeaderNavContext()

  return buildTopNavLinks({ modules, docsLink, isAuthed, t })
}

export function useTopNavLinkSlots(): TopNavLinkSlots {
  const { modules, docsLink, isAuthed, t } = useHeaderNavContext()

  return buildTopNavLinkSlots({ modules, docsLink, isAuthed, t })
}
