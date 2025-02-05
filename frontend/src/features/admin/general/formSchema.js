import * as z from 'zod'

export const formSchema = z.object({
  site_name: z
    .string({
      required_error: 'Site name is required.'
    })
    .min(1, {
      message: 'Site name must be at least 1 characters.'
    }),
  lang: z.string().optional(),
  timezone: z.string().optional(),
  business_hours_id: z.string().optional(),
  logo_url: z.string().url({
    message: 'Logo URL must be a valid URL.'
  }).url().optional(),
  root_url: z
    .string({
      required_error: 'Root URL is required.'
    })
    .url({
      message: 'Root URL must be a valid URL.'
    }).url(),
  favicon_url: z
    .string({
      required_error: 'Favicon URL is required.'
    })
    .url({
      message: 'Favicon URL must be a valid URL.'
    }).url(),
  max_file_upload_size: z
    .number({
      required_error: 'Max upload file size is required.'
    })
    .min(1, {
      message: 'Max upload file size must be at least 1 MB.'
    })
    .max(30, {
      message: 'Max upload file size cannot exceed 30 MB.'
    }),
  allowed_file_upload_extensions: z.array(z.string()).nullable().default([]).optional()
})
