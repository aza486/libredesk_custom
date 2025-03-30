import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('form.error.name.required'),
    })
    .min(3, {
      message: t('admin.conversation_tags.name.valid'),
    })
})
