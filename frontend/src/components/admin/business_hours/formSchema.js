import * as z from 'zod'

export const formSchema = z.object({
    name: z
        .string({
            required_error: 'Name is required.'
        })
        .min(1, {
            message: 'Name must be at least 1 character.'
        }),
    description: z.string().optional(),
    is_always_open: z.string().default('true').optional(),
})
