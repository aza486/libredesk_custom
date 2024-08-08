import * as z from 'zod'

export const oidcLoginFormSchema = z.object({
    provider_url: z
        .string({
            required_error: 'Provider URL is required.'
        })
        .url({
            message: 'Provider URL must be a valid URL.'
        }),
    client_id: z
        .string({
            required_error: 'Client ID is required.'
        }),
    client_secret: z
        .string({
            required_error: 'Client Secret is required.'
        }),
    redirect_uri: z
        .string({
            required_error: 'Redirect URI is required.'
        })
        .url({
            message: 'Redirect URI must be a valid URL.'
        })
})
