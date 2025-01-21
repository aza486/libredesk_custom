import * as z from 'zod'

export const teamFormSchema = z.object({
  name: z
    .string({
      required_error: 'Team name is required.'
    })
    .min(2, {
      message: 'Team name must be at least 2 characters.'
    }),
  emoji: z.string({ required_error: 'Emoji is required.' }),
  conversation_assignment_type: z.string({ required_error: 'Conversation assignment type is required.' }),
  timezone: z.string({ required_error: 'Timezone is required.' }),
  business_hours_id: z.number().optional().nullable(),
})
