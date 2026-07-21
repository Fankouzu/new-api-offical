import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  MODEL_HERO_SLIDES,
  getNextModelSlideIndex,
} from './custom-model-hero-data'

export function CustomModelHero() {
  const { t } = useTranslation()
  const [activeIndex, setActiveIndex] = useState(0)

  useEffect(() => {
    let timer: number | undefined

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
        setActiveIndex((current) =>
          getNextModelSlideIndex(current, MODEL_HERO_SLIDES.length)
        )
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
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [])

  const activeSlide = MODEL_HERO_SLIDES[activeIndex]

  return (
    <section
      aria-roledescription='carousel'
      aria-label={t('Featured AI models')}
      className='relative h-[clamp(340px,38vw,500px)] overflow-hidden border-y border-white/10 bg-black'
    >
      {MODEL_HERO_SLIDES.map((slide, index) => (
        <img
          key={slide.name}
          src={slide.image}
          alt=''
          aria-hidden
          fetchPriority={index === 0 ? 'high' : 'auto'}
          className={`absolute inset-0 size-full object-cover transition-opacity duration-2000 ease-in-out motion-reduce:transition-none ${
            index === activeIndex ? 'opacity-100' : 'opacity-0'
          }`}
        />
      ))}

      <div className='absolute inset-0 bg-[linear-gradient(90deg,rgba(4,8,12,0.88)_0%,rgba(4,8,12,0.68)_38%,rgba(4,8,12,0.12)_72%,rgba(4,8,12,0.24)_100%)]' />

      <div className='relative z-10 flex h-full items-center px-[clamp(24px,6vw,96px)]'>
        <div className='max-w-xl text-white'>
          <p className='mb-3 text-xs font-semibold tracking-[0.18em] text-white/55 uppercase'>
            {t('Featured model')}
          </p>
          <h1 className='text-[clamp(2.5rem,5vw,4.75rem)] leading-none font-bold tracking-tight'>
            {activeSlide.name}
          </h1>
          <p className='mt-5 max-w-lg text-sm leading-relaxed text-white/72 sm:text-base'>
            {t(activeSlide.description)}
          </p>
          <div className='mt-6 flex flex-wrap gap-2'>
            {activeSlide.tags.map((tag) => (
              <span
                key={tag}
                className='rounded-md border border-white/18 bg-black/22 px-3 py-1.5 text-xs font-medium text-white/78 backdrop-blur-md'
              >
                {t(tag)}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className='absolute right-6 bottom-5 z-20 flex items-center gap-2'>
        {MODEL_HERO_SLIDES.map((slide, index) => (
          <button
            key={slide.name}
            type='button'
            aria-label={t('Show {{model}}', { model: slide.name })}
            aria-current={index === activeIndex ? 'true' : undefined}
            onClick={() => setActiveIndex(index)}
            className={`h-1.5 rounded-full transition-[width,background-color] duration-300 ${
              index === activeIndex
                ? 'w-8 bg-white'
                : 'w-3 bg-white/35 hover:bg-white/65'
            }`}
          />
        ))}
      </div>
    </section>
  )
}
