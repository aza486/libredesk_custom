import * as z from 'zod'

export const s3FormSchema = z.object({
    provider: z
        .string({
            required_error: 'Provider is required.'
        }),
    region: z
        .string({
            required_error: 'Region is required.'
        }),
    access_key: z
        .string({
            required_error: 'AWS access key is required.'
        }),
    access_secret: z
        .string({
            required_error: 'AWS access secret is required.'
        }),
    bucket_type: z
        .string({
            required_error: 'Bucket type is required.'
        }),
    bucket: z
        .string({
            required_error: 'Bucket is required.'
        }),
    bucket_path: z
        .string({
            required_error: 'Bucket path is required.'
        }),
    upload_expiry: z
        .string({
            required_error: 'Upload expiry is required.'
        }),
    url: z
        .string({
            required_error: 'S3 backend URL is required.'
        })
        .url({
            message: 'S3 backend URL must be a valid URL.'
        }),
})


export const localFsFormSchema = z.object({
    provider: z
        .string({
            required_error: 'Provider is required.'
        }),
    upload_path: z
        .string({
            required_error: 'Upload path is required.'
        }),
})