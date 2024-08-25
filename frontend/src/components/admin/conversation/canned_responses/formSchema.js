import * as z from 'zod'

export const formSchema = z.object({
  title: z
    .string({
      required_error: 'Title is required.'
    })
    .min(1, {
      message: 'Title must be at least 1 character.'
    }),
  content: z
    .string({
      required_error: 'Content is required.'
    })
    .min(1, {
      message: 'Content must be atleast 3 characters.'
    })
})
