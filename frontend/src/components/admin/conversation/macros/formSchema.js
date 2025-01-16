import * as z from 'zod'

const actionSchema = z.array(
  z.object({
    type: z.string().min(1, 'Action type required'),
    value: z.array(z.string().min(1, 'Action value required')).min(1, 'Action value required'),
  })
)

export const formSchema = z.object({
  name: z.string().min(1, 'Macro name is required'),
  message_content: z.string().optional(),
  actions: actionSchema,
  visibility: z.enum(['all', 'team', 'user']),
  team_id: z.string().nullable().optional(),
  user_id: z.string().nullable().optional(),
})