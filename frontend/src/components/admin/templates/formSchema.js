import * as z from 'zod'

export const formSchema = z.object({
    name: z
        .string({
            required_error: 'Template name is required.'
        }),
    body: z
        .string({
            required_error: 'Template body is required.'
        }),
    is_default: z.boolean().optional()
})
