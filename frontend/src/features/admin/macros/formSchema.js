import * as z from 'zod'
import { getTextFromHTML } from '@/utils/strings.js'

const actionSchema = z.array(
  z.object({
    type: z.string().min(1, 'Action type required'),
    value: z.array(z.string().min(1, 'Action value required')).min(1, 'Action value required'),
  })
)

export const formSchema = z.object({
  name: z.string().min(1, 'Macro name is required'),
  message_content: z.string().optional(),
  actions: actionSchema.optional().default([]), // Default to empty array if not provided
  visibility: z.enum(['all', 'team', 'user']),
  team_id: z.string().nullable().optional(),
  user_id: z.string().nullable().optional(),
})
  .refine(
    (data) => {
      // Check if message_content has non-empty text after stripping HTML
      const hasMessageContent = getTextFromHTML(data.message_content || '').trim().length > 0
      // Check if actions has at least one valid action
      const hasValidActions = data.actions && data.actions.length > 0
      // Either message content or actions must be valid
      return hasMessageContent || hasValidActions
    },
    {
      message: 'Either message content or actions are required',
      // Field path to highlight
      path: ['message_content'],
    }
  )
  .refine(
    (data) => {
      // If visibility is 'team', team_id is required
      if (data.visibility === 'team' && !data.team_id) {
        return false
      }
      // If visibility is 'user', user_id is required
      if (data.visibility === 'user' && !data.user_id) {
        return false
      }
      // Otherwise, validation passes
      return true
    },
    {
      message: 'team is required when visibility is "team", and user is required when visibility is "user"',
      // Field path to highlight
      path: ['visibility'],
    }
  )