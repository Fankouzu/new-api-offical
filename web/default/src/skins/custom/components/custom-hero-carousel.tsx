import { createElement, useEffect, useRef, useState } from 'react'

export const CUSTOM_HERO_IMAGES = [
  'https://image.lizh.ai/e0f4935c-a82c-41aa-e917-0e90e1eac700/large',
  'https://image.lizh.ai/35f940d0-4411-4abc-a1bb-74ed7a80cb00/large',
  'https://image.lizh.ai/135c21a5-cb53-406e-42a6-2646fd077d00/large',
  'https://image.lizh.ai/060453f7-cb74-43ec-1f38-cd0c9bfcf900/large',
] as const

export function getNextHeroSlideIndex(current: number, total: number) {
  return (current + 1) % total
}

export function CustomHeroCarousel() {
  const [activeIndex, setActiveIndex] = useState(0)
  const [exitingIndex, setExitingIndex] = useState<number | null>(null)
  const activeIndexRef = useRef(0)

  useEffect(() => {
    let timer: number | undefined
    let exitTimer: number | undefined

    const stop = () => {
      if (timer !== undefined) {
        window.clearInterval(timer)
        timer = undefined
      }
    }

    const start = () => {
      stop()
      if (document.hidden) return

      timer = window.setInterval(() => {
        const current = activeIndexRef.current
        const next = getNextHeroSlideIndex(current, CUSTOM_HERO_IMAGES.length)

        if (exitTimer !== undefined) window.clearTimeout(exitTimer)
        setExitingIndex(current)
        activeIndexRef.current = next
        setActiveIndex(next)
        exitTimer = window.setTimeout(() => setExitingIndex(null), 1000)
      }, 5000)
    }

    const handleVisibilityChange = () => {
      if (document.hidden) stop()
      else start()
    }

    start()
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      stop()
      if (exitTimer !== undefined) window.clearTimeout(exitTimer)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [])

  return createElement(
    'div',
    {
      'aria-hidden': true,
      className: 'absolute inset-0 overflow-hidden bg-black',
    },
    ...CUSTOM_HERO_IMAGES.map((src, index) =>
      createElement('img', {
        key: src,
        src,
        alt: '',
        fetchPriority: index === 0 ? 'high' : 'auto',
        className: `custom-hero-slide absolute inset-0 size-full object-cover transition-opacity duration-2000 ease-in-out motion-reduce:transition-none ${
          index === activeIndex || index === exitingIndex ? 'is-moving' : ''
        } ${index === activeIndex ? 'opacity-100' : 'opacity-0'}`,
      })
    ),
    createElement('div', { className: 'absolute inset-0 bg-black/55' })
  )
}
