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
import { useEffect, useMemo } from 'react'
import * as z from 'zod'
import { useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  HEADER_NAV_DEFAULT,
  type HeaderNavCustomLinkPosition,
  type HeaderNavModulesConfig,
  serializeHeaderNavModules,
} from './config'

const customLinkSchema = z.object({
  id: z.string().min(1),
  title: z.string().trim().min(1, 'Title is required'),
  href: z
    .string()
    .trim()
    .min(1, 'URL is required')
    .refine(
      (value) => {
        if (value.startsWith('/')) return !value.startsWith('//')
        try {
          const url = new URL(value)
          return url.protocol === 'http:' || url.protocol === 'https:'
        } catch {
          return false
        }
      },
      { message: 'Use an http(s) URL or a local path starting with /' }
    ),
  enabled: z.boolean(),
  external: z.boolean(),
  requireAuth: z.boolean(),
  position: z.enum([
    'after_console',
    'after_pricing',
    'after_rankings',
    'after_docs',
    'end',
    'before_search',
    'after_search',
    'before_notifications',
    'after_notifications',
    'before_theme',
    'after_theme',
    'before_language',
    'after_language',
  ]),
  display: z.enum(['text', 'icon']),
  icon: z.string().optional(),
})

const headerNavSchema = z.object({
  home: z.boolean(),
  console: z.boolean(),
  pricingEnabled: z.boolean(),
  pricingRequireAuth: z.boolean(),
  rankingsEnabled: z.boolean(),
  rankingsRequireAuth: z.boolean(),
  docs: z.boolean(),
  about: z.boolean(),
  customLinks: z.array(customLinkSchema),
})

type HeaderNavFormValues = z.infer<typeof headerNavSchema>
type HeaderNavBooleanField = Extract<
  {
    [Key in keyof HeaderNavFormValues]: HeaderNavFormValues[Key] extends boolean
      ? Key
      : never
  }[keyof HeaderNavFormValues],
  string
>
type HeaderNavSimpleField = Extract<
  HeaderNavBooleanField,
  'home' | 'console' | 'docs' | 'about'
>

type HeaderNavigationSectionProps = {
  config: HeaderNavModulesConfig
  initialSerialized: string
}

const toFormValues = (config: HeaderNavModulesConfig): HeaderNavFormValues => ({
  home:
    config.home === undefined ? HEADER_NAV_DEFAULT.home : Boolean(config.home),
  console:
    config.console === undefined
      ? HEADER_NAV_DEFAULT.console
      : Boolean(config.console),
  pricingEnabled:
    config.pricing?.enabled === undefined
      ? HEADER_NAV_DEFAULT.pricing.enabled
      : Boolean(config.pricing.enabled),
  pricingRequireAuth:
    config.pricing?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.pricing.requireAuth
      : Boolean(config.pricing.requireAuth),
  rankingsEnabled:
    config.rankings?.enabled === undefined
      ? HEADER_NAV_DEFAULT.rankings.enabled
      : Boolean(config.rankings.enabled),
  rankingsRequireAuth:
    config.rankings?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.rankings.requireAuth
      : Boolean(config.rankings.requireAuth),
  docs:
    config.docs === undefined ? HEADER_NAV_DEFAULT.docs : Boolean(config.docs),
  about:
    config.about === undefined
      ? HEADER_NAV_DEFAULT.about
      : Boolean(config.about),
  customLinks: config.customLinks.map((link) => ({ ...link })),
})

const customLinkPositions: Array<{
  value: HeaderNavCustomLinkPosition
  label: string
}> = [
  { value: 'after_console', label: 'After Console' },
  { value: 'after_pricing', label: 'After Model Square' },
  { value: 'after_rankings', label: 'After Rankings' },
  { value: 'after_docs', label: 'After Docs' },
  { value: 'end', label: 'After About' },
  { value: 'before_search', label: 'Before Search' },
  { value: 'after_search', label: 'After Search' },
  { value: 'before_notifications', label: 'Before Notifications' },
  { value: 'after_notifications', label: 'After Notifications' },
  { value: 'before_theme', label: 'Before Theme' },
  { value: 'after_theme', label: 'After Theme' },
  { value: 'before_language', label: 'Before Language' },
  { value: 'after_language', label: 'After Language' },
]

const customLinkDisplays = [
  { value: 'text', label: 'Text' },
  { value: 'icon', label: 'Icon' },
]

const customLinkIcons = [{ value: 'telegram', label: 'Telegram' }]

export function HeaderNavigationSection({
  config,
  initialSerialized,
}: HeaderNavigationSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formDefaults = useMemo(() => toFormValues(config), [config])

  const form = useForm<HeaderNavFormValues>({
    resolver: zodResolver(headerNavSchema),
    defaultValues: formDefaults,
  })
  const customLinksFieldArray = useFieldArray({
    control: form.control,
    name: 'customLinks',
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [formDefaults, form])

  const onSubmit = async (values: HeaderNavFormValues) => {
    const payload: HeaderNavModulesConfig = {
      ...config,
      home: values.home,
      console: values.console,
      docs: values.docs,
      about: values.about,
      pricing: {
        ...(config.pricing ?? HEADER_NAV_DEFAULT.pricing),
        enabled: values.pricingEnabled,
        requireAuth: values.pricingRequireAuth,
      },
      rankings: {
        ...(config.rankings ?? HEADER_NAV_DEFAULT.rankings),
        enabled: values.rankingsEnabled,
        requireAuth: values.rankingsRequireAuth,
      },
      customLinks: values.customLinks.map((link) => ({
        ...link,
        id: link.id.trim(),
        title: link.title.trim(),
        href: link.href.trim(),
        icon:
          link.icon?.trim() ||
          (link.display === 'icon' ? 'telegram' : undefined),
      })),
    }

    const serialized = serializeHeaderNavModules(payload)
    if (serialized === initialSerialized) {
      return
    }

    await updateOption.mutateAsync({
      key: 'HeaderNavModules',
      value: serialized,
    })
  }

  const resetToDefault = () => {
    form.reset(toFormValues(HEADER_NAV_DEFAULT))
  }

  const addTelegramLink = () => {
    customLinksFieldArray.append({
      id: `custom-${Date.now()}`,
      title: 'Telegram',
      href: 'https://t.me/',
      enabled: true,
      external: true,
      requireAuth: false,
      position: 'end',
      display: 'icon',
      icon: 'telegram',
    })
  }

  const simpleModules: Array<{
    key: HeaderNavSimpleField
    title: string
    description: string
  }> = [
    {
      key: 'home',
      title: t('Home'),
      description: t('Landing page with system overview.'),
    },
    {
      key: 'console',
      title: t('Console'),
      description: t('User dashboard and quota controls.'),
    },
    {
      key: 'docs',
      title: t('Docs'),
      description: t('Documentation or external knowledge base.'),
    },
    {
      key: 'about',
      title: t('About'),
      description: t('Static page describing the platform.'),
    },
  ]

  const accessModules: Array<{
    enabledKey: HeaderNavBooleanField
    requireAuthKey: HeaderNavBooleanField
    requireAuthDependsOn: 'pricingEnabled' | 'rankingsEnabled'
    title: string
    description: string
    requireAuthTitle: string
    requireAuthDescription: string
  }> = [
    {
      enabledKey: 'pricingEnabled',
      requireAuthKey: 'pricingRequireAuth',
      requireAuthDependsOn: 'pricingEnabled',
      title: t('Model Square'),
      description: t('Public model catalog and pricing page.'),
      requireAuthTitle: t('Require login to view models'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the pricing directory.'
      ),
    },
    {
      enabledKey: 'rankingsEnabled',
      requireAuthKey: 'rankingsRequireAuth',
      requireAuthDependsOn: 'rankingsEnabled',
      title: t('Rankings'),
      description: t('Public rankings page based on live usage data.'),
      requireAuthTitle: t('Require login to view rankings'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the rankings page.'
      ),
    },
  ]

  return (
    <SettingsSection
      title={t('Header navigation')}
      description={t('Enable or disable top navigation modules globally.')}
    >
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-6'>
          <div className='grid gap-4 md:grid-cols-2'>
            {simpleModules.map((module) => (
              <FormField
                key={module.key}
                control={form.control}
                name={module.key}
                render={({ field }) => (
                  <FormItem className='flex flex-row items-start justify-between rounded-lg border p-4'>
                    <div className='space-y-0.5 pe-4'>
                      <FormLabel className='text-base'>
                        {module.title}
                      </FormLabel>
                      <FormDescription>{module.description}</FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ))}
          </div>

          <div className='grid gap-4 lg:grid-cols-2'>
            {accessModules.map((module) => (
              <div key={module.enabledKey} className='rounded-lg border p-4'>
                <FormField
                  control={form.control}
                  name={module.enabledKey}
                  render={({ field }) => (
                    <FormItem className='flex flex-row items-start justify-between rounded-lg border p-4'>
                      <div className='space-y-0.5 pe-4'>
                        <FormLabel className='text-base'>
                          {module.title}
                        </FormLabel>
                        <FormDescription>{module.description}</FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name={module.requireAuthKey}
                  render={({ field }) => (
                    <FormItem className='mt-4 flex flex-row items-start justify-between rounded-lg border border-dashed p-4'>
                      <div className='space-y-0.5 pe-4'>
                        <FormLabel className='text-base'>
                          {module.requireAuthTitle}
                        </FormLabel>
                        <FormDescription>
                          {module.requireAuthDescription}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          disabled={!form.watch(module.requireAuthDependsOn)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            ))}
          </div>

          <div className='space-y-4 rounded-lg border p-4'>
            <div className='flex flex-wrap items-start justify-between gap-3'>
              <div className='space-y-1'>
                <h3 className='text-base font-medium'>
                  {t('Custom navigation links')}
                </h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Add optional links such as Telegram, status pages, or community channels per site.'
                  )}
                </p>
              </div>
              <Button type='button' variant='outline' onClick={addTelegramLink}>
                <Plus data-icon='inline-start' />
                {t('Add social link')}
              </Button>
            </div>

            {customLinksFieldArray.fields.length === 0 ? (
              <div className='text-muted-foreground rounded-lg border border-dashed p-4 text-sm'>
                {t('No custom navigation links configured.')}
              </div>
            ) : (
              <div className='space-y-3'>
                {customLinksFieldArray.fields.map((field, index) => (
                  <div key={field.id} className='rounded-lg border p-4'>
                    <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_180px_140px_140px_auto]'>
                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.title`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Title')}</FormLabel>
                            <FormControl>
                              <Input placeholder='Telegram' {...field} />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.href`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('URL')}</FormLabel>
                            <FormControl>
                              <Input
                                placeholder='https://t.me/your_channel'
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.position`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Position')}</FormLabel>
                            <Select
                              value={field.value}
                              onValueChange={field.onChange}
                            >
                              <FormControl>
                                <SelectTrigger className='w-full'>
                                  <SelectValue />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                <SelectGroup>
                                  {customLinkPositions.map((position) => (
                                    <SelectItem
                                      key={position.value}
                                      value={position.value}
                                    >
                                      {t(position.label)}
                                    </SelectItem>
                                  ))}
                                </SelectGroup>
                              </SelectContent>
                            </Select>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.display`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Display')}</FormLabel>
                            <Select
                              value={field.value}
                              onValueChange={field.onChange}
                            >
                              <FormControl>
                                <SelectTrigger className='w-full'>
                                  <SelectValue />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                <SelectGroup>
                                  {customLinkDisplays.map((display) => (
                                    <SelectItem
                                      key={display.value}
                                      value={display.value}
                                    >
                                      {t(display.label)}
                                    </SelectItem>
                                  ))}
                                </SelectGroup>
                              </SelectContent>
                            </Select>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.icon`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Icon')}</FormLabel>
                            <Select
                              value={field.value || 'telegram'}
                              onValueChange={field.onChange}
                            >
                              <FormControl>
                                <SelectTrigger className='w-full'>
                                  <SelectValue />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                <SelectGroup>
                                  {customLinkIcons.map((icon) => (
                                    <SelectItem
                                      key={icon.value}
                                      value={icon.value}
                                    >
                                      {t(icon.label)}
                                    </SelectItem>
                                  ))}
                                </SelectGroup>
                              </SelectContent>
                            </Select>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <div className='flex items-end'>
                        <Button
                          type='button'
                          variant='outline'
                          size='icon'
                          onClick={() => customLinksFieldArray.remove(index)}
                          aria-label={t('Delete custom navigation link')}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </div>

                    <div className='mt-4 grid gap-4 md:grid-cols-3'>
                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.enabled`}
                        render={({ field }) => (
                          <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                            <FormLabel>{t('Enabled')}</FormLabel>
                            <FormControl>
                              <Switch
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.external`}
                        render={({ field }) => (
                          <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                            <FormLabel>{t('Open in new tab')}</FormLabel>
                            <FormControl>
                              <Switch
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`customLinks.${index}.requireAuth`}
                        render={({ field }) => (
                          <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                            <FormLabel>{t('Require login')}</FormLabel>
                            <FormControl>
                              <Switch
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className='flex flex-wrap gap-3'>
            <Button type='button' variant='outline' onClick={resetToDefault}>
              {t('Reset to default')}
            </Button>
            <Button type='submit' disabled={updateOption.isPending}>
              {updateOption.isPending ? t('Saving...') : t('Save navigation')}
            </Button>
          </div>
        </form>
      </Form>
    </SettingsSection>
  )
}
