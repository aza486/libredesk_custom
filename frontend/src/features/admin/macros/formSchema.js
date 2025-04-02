import * as z from 'zod'
import { getTextFromHTML } from '@/utils/strings.js'

const actionSchema = (t) => z.array(
  z.object({
    type: z.string().min(1, t('admin.macro.actionTypeRequired')),
    value: z.array(z.string().min(1, t('admin.macro.actionValueRequired'))),
  })
)

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  message_content: z.string().optional(),
  actions: actionSchema(t).optional().default([]),
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
      message: t('admin.macro.messageOrActionRequired'),
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
      message: t('admin.macro.teamOrUserRequired'),
      // Field path to highlight
      path: ['visibility'],
    }
  )