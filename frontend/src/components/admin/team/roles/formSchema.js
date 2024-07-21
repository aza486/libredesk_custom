import * as z from 'zod'

export const formSchema = z.object({
    name: z
        .string({
            required_error: 'Name is required.'
        })
        .min(2, {
            message: 'First name must be at least 2 characters.'
        }),

    description: z
        .string({
            required_error: 'Description is required.'
        })
        .min(2, {
            message: 'First name must be at least 2 characters.'
        }),
    permissions: z.array(z.string()).optional()
})