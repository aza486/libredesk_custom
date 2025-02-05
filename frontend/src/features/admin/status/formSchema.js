import * as z from 'zod'

export const formSchema = z.object({
  name: z
    .string({
      required_error: 'Status name is required.'
    })
    .min(1, {
      message: 'Status must be at least 1 character.'
    })
})
