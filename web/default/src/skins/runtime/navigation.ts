import type { SkinNavigationLink } from './contracts'
import { assertSkinRouteAllowed } from './route-policy'

type HeaderLink = {
  title: string
  href: string
  disabled?: boolean
  external?: boolean
  display?: 'text' | 'icon'
  icon?: string
}

export function mergeSkinHeaderLinks<T extends HeaderLink>(
  hostLinks: readonly T[],
  contributions: readonly SkinNavigationLink[],
  allowedPaths: readonly string[],
  t: (key: string) => string
): Array<T | HeaderLink> {
  const result: Array<T | HeaderLink> = [...hostLinks]
  const allowedPathSet = new Set(allowedPaths)
  const hostPaths = new Set(hostLinks.map((link) => link.href))
  const contributionPaths = new Set<string>()

  for (const contribution of contributions) {
    if (contributionPaths.has(contribution.href)) {
      throw new Error(
        `Duplicate skin header navigation contribution: ${contribution.href}`
      )
    }
    contributionPaths.add(contribution.href)

    assertSkinRouteAllowed(contribution.href)

    if (!allowedPathSet.has(contribution.href)) {
      throw new Error(
        `Skin header navigation path is not declared in active build routes: ${contribution.href}`
      )
    }

    if (hostPaths.has(contribution.href)) {
      throw new Error(
        `Skin header navigation conflicts with a host link: ${contribution.href}`
      )
    }
  }

  const sortedContributions = [...contributions].sort(
    (left, right) =>
      (left.order ?? Number.POSITIVE_INFINITY) -
        (right.order ?? Number.POSITIVE_INFINITY) ||
      left.href.localeCompare(right.href)
  )

  for (const contribution of sortedContributions) {
    result.push({ title: t(contribution.labelKey), href: contribution.href })
  }

  return result
}
