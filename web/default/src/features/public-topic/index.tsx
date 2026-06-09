import { PublicLayout } from '@/components/layout'
import { getPublicTopic, type PublicTopic } from './topics'

type PublicTopicPageProps = {
  path: string
}

export function PublicTopicPage({ path }: PublicTopicPageProps) {
  const topic = getPublicTopic(path)

  if (!topic) {
    return null
  }

  return (
    <PublicLayout>
      <article className='mx-auto max-w-5xl px-4 py-12 sm:py-16'>
        <header className='max-w-3xl space-y-4'>
          <p className='text-muted-foreground text-sm font-medium tracking-wide uppercase'>
            Lizh AI
          </p>
          <h1 className='text-3xl font-bold tracking-tight sm:text-5xl'>
            {topic.h1}
          </h1>
          <p className='text-muted-foreground text-base leading-7 sm:text-lg'>
            {topic.intro}
          </p>
        </header>

        <div className='mt-10 grid gap-8 lg:grid-cols-[minmax(0,1fr)_280px]'>
          <div className='space-y-8'>
            {topic.sections.map((section) => (
              <section key={section.title} className='space-y-3'>
                <h2 className='text-2xl font-semibold tracking-tight'>
                  {section.title}
                </h2>
                <p className='text-muted-foreground leading-7'>
                  {section.body}
                </p>
              </section>
            ))}

            <section className='space-y-4'>
              <h2 className='text-2xl font-semibold tracking-tight'>FAQ</h2>
              <div className='divide-border divide-y rounded-lg border'>
                {topic.faqs.map((faq) => (
                  <div key={faq.question} className='space-y-2 p-5'>
                    <h3 className='font-medium'>{faq.question}</h3>
                    <p className='text-muted-foreground text-sm leading-6'>
                      {faq.answer}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          </div>

          <aside className='space-y-3 lg:sticky lg:top-20 lg:self-start'>
            <h2 className='text-sm font-semibold tracking-wide uppercase'>
              Continue exploring
            </h2>
            <nav className='flex flex-col gap-2'>
              {topic.links.map((link) => (
                <TopicLink key={link.href} link={link} />
              ))}
            </nav>
          </aside>
        </div>
      </article>
    </PublicLayout>
  )
}

function TopicLink({ link }: { link: PublicTopic['links'][number] }) {
  return (
    <a
      className='border-border bg-background hover:bg-muted hover:text-foreground inline-flex h-8 items-center justify-start rounded-lg border px-3 text-sm font-medium transition-colors'
      href={link.href}
    >
      {link.label}
    </a>
  )
}
