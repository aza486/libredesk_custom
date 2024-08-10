import * as z from 'zod'

export const teamFormSchema = z.object({
  name: z
    .string({
      required_error: 'Team name is required.'
    })
    .min(2, {
      message: 'Team name must be at least 2 characters.'
    }),
  auto_assign_conversations: z.boolean().optional()
})
