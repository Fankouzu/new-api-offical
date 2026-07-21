import z from 'zod'

export const pricingSearchSchema = z.object({
  search: z.string().optional(),
  sort: z.string().optional(),
  vendor: z.string().optional(),
  group: z.string().optional(),
  quotaType: z.string().optional(),
  endpointType: z.string().optional(),
  tag: z.string().optional(),
  tokenUnit: z.enum(['M', 'K']).optional(),
  view: z.enum(['card', 'table']).optional().catch(undefined),
  rechargePrice: z.boolean().optional(),
})

export type PricingSearch = z.infer<typeof pricingSearchSchema>

export function parsePricingSearch(search: unknown): PricingSearch {
  const result = pricingSearchSchema.safeParse(search)
  return result.success ? result.data : {}
}
