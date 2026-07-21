import type {
  SkinRouteBuildDefinition,
  SkinRouteNavigation,
} from './build-contracts'

type HeaderLink = {
  title: string
  href: string
  disabled?: boolean
  external?: boolean
  display?: 'text' | 'icon'
  icon?: string
}

function isValidHeaderNavigation(
  navigation: SkinRouteNavigation | undefined
): navigation is SkinRouteNavigation & { position: 'header' } {
  return (
    navigation?.position === 'header' &&
    typeof navigation.labelKey === 'string' &&
    navigation.labelKey.trim() !== '' &&
    (navigation.order === undefined ||
      (Number.isInteger(navigation.order) && navigation.order >= 0))
  )
}

export function mergeSkinHeaderLinks<T extends HeaderLink>(
  hostLinks: readonly T[],
  routes: readonly SkinRouteBuildDefinition[],
  t: (key: string) => string
): Array<T | HeaderLink> {
  const result: Array<T | HeaderLink> = [...hostLinks]
  const occupiedPaths = new Set(hostLinks.map((link) => link.href))
  const contributions = routes
    .filter((route) => isValidHeaderNavigation(route.navigation))
    .sort(
      (left, right) =>
        (left.navigation?.order ?? Number.POSITIVE_INFINITY) -
          (right.navigation?.order ?? Number.POSITIVE_INFINITY) ||
        left.path.localeCompare(right.path)
    )

  for (const route of contributions) {
    if (
      occupiedPaths.has(route.path) ||
      !isValidHeaderNavigation(route.navigation)
    ) {
      continue
    }

    occupiedPaths.add(route.path)
    result.push({ title: t(route.navigation.labelKey), href: route.path })
  }

  return result
}
