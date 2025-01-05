import * as z from 'zod'
import { isGoHourMinuteDuration } from '@/utils/strings'

export const formSchema = z.object({
    name: z
        .string({
            required_error: 'Name is required.'
        })
        .max(255, {
            message: 'Name must be at most 255 characters.'
        }),
    description: z
        .string()
        .max(255, {
            message: 'Description must be at most 255 characters.'
        })
        .optional(),
    first_response_time: z.string().optional().refine(isGoHourMinuteDuration, {
        message:
            'Invalid duration format. Should be a number followed by h (hours), m (minutes).'
    }),
    resolution_time: z.string().optional().refine(isGoHourMinuteDuration, {
        message:
            'Invalid duration format. Should be a number followed by h (hours), m (minutes).'
    }),
})
