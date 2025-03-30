import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  site_name: z
    .string({
      required_error: t('admin.general.site_name.required'),
    })
    .min(1, {
      message: t('admin.general.site_name.min'),
    }),
  lang: z.string().optional(),
  timezone: z.string().optional(),
  business_hours_id: z.string().optional(),
  logo_url: z.string().url({
    message: t('admin.general.logo_url.valid'),
  }).or(z.literal(''))
    .optional(),
  root_url: z
    .string({
      required_error: t('admin.general.root_url.required')
    })
    .url({
      message: t('admin.general.root_url.valid')
    }).url(),
  favicon_url: z
    .string({
      required_error: t('admin.general.favicon_url.required')
    })
    .url({
      message: t('admin.general.favicon_url.valid')
    }).url(),
  max_file_upload_size: z
    .number({
      required_error: t('admin.general.max_allowed_file_upload_size.required')
    })
    .min(1, {
      message: t('admin.general.max_allowed_file_upload_size.valid')
    })
    .max(500, {
      message: t('admin.general.max_allowed_file_upload_size.valid')
    }),
  allowed_file_upload_extensions: z.array(z.string()).nullable().default([]).optional()
})
