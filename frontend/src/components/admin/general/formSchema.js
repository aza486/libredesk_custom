import * as z from 'zod'

export const formSchema = z.object({
    site_name: z
        .string({
            required_error: 'Site name is required.'
        })
        .min(1, {
            message: 'Site name must be at least 1 characters.'
        }),
    lang: z
        .string().optional(),
    root_url: z
        .string({
            required_error: 'Root URL is required.'
        })
        .url({
            message: 'Root URL must be a valid URL.'
        }),
    favicon_url: z
        .string({
            required_error: 'Favicon URL is required.'
        })
        .url({
            message: 'Favicon URL must be a valid URL.'
        })
})
