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
import { useEffect, useId, useRef } from 'react'

export type TelegramAuthPayload = {
  id?: number | string
  first_name?: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date?: number | string
  hash?: string
  lang?: string
}

type TelegramLoginWidgetProps = {
  botName: string
  mode?: 'callback' | 'redirect'
  authUrl?: string
  onAuth?: (payload: TelegramAuthPayload) => void
  size?: 'large' | 'medium' | 'small'
  radius?: number
  requestAccess?: 'write'
  className?: string
}

declare global {
  interface Window {
    TelegramLoginWidget?: Record<string, (user: TelegramAuthPayload) => void>
  }
}

export function TelegramLoginWidget({
  botName,
  mode = 'callback',
  authUrl,
  onAuth,
  size = 'large',
  radius = 8,
  requestAccess = 'write',
  className,
}: TelegramLoginWidgetProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const id = useId().replace(/:/g, '')

  useEffect(() => {
    const container = containerRef.current
    if (!container || !botName) return

    const callbackName = `onAuth${id}`
    window.TelegramLoginWidget = window.TelegramLoginWidget || {}
    if (mode === 'callback' && onAuth) {
      window.TelegramLoginWidget[callbackName] = onAuth
    }

    container.innerHTML = ''
    const script = document.createElement('script')
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.async = true
    script.setAttribute('data-telegram-login', botName)
    script.setAttribute('data-size', size)
    script.setAttribute('data-radius', String(radius))
    script.setAttribute('data-request-access', requestAccess)
    if (mode === 'redirect' && authUrl) {
      script.setAttribute('data-auth-url', authUrl)
    } else {
      script.setAttribute(
        'data-onauth',
        `TelegramLoginWidget.${callbackName}(user)`
      )
    }

    container.appendChild(script)

    return () => {
      container.innerHTML = ''
      if (window.TelegramLoginWidget) {
        delete window.TelegramLoginWidget[callbackName]
      }
    }
  }, [authUrl, botName, id, mode, onAuth, radius, requestAccess, size])

  return <div ref={containerRef} className={className} />
}
