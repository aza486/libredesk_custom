import * as z from 'zod'
import { isGoHourMinuteDuration } from '@/utils/strings'

export const formSchema = z.object({
    name: z
        .string()
        .min(1, { message: 'Name is required' })
        .max(255, { message: 'Name must be at most 255 characters.' }),
    description: z
        .string()
        .min(1, { message: 'Description is required' })
        .max(255, { message: 'Description must be at most 255 characters.' }),
    first_response_time: z.string().refine(isGoHourMinuteDuration, {
        message:
            'Invalid duration format. Should be a number followed by h (hours), m (minutes).'
    }),
    resolution_time: z.string().refine(isGoHourMinuteDuration, {
        message:
            'Invalid duration format. Should be a number followed by h (hours), m (minutes).'
    }),
    notifications: z
        .array(
            z
                .object({
                    type: z.enum(['breach', 'warning']),
                    time_delay_type: z.enum(['immediately', 'after', 'before']),
                    time_delay: z.string().optional(),
                    recipients: z
                        .array(z.string())
                        .min(1, { message: 'At least one recipient is required' })
                })
                .superRefine((obj, ctx) => {
                    if (obj.time_delay_type !== 'immediately') {
                        if (!obj.time_delay || obj.time_delay === '') {
                            ctx.addIssue({
                                code: z.ZodIssueCode.custom,
                                message:
                                    'Delay is required',
                                path: ['time_delay']
                            })
                        } else if (!isGoHourMinuteDuration(obj.time_delay)) {
                            ctx.addIssue({
                                code: z.ZodIssueCode.custom,
                                message:
                                    'Invalid duration format. Should be a number followed by h (hours), m (minutes).',
                                path: ['time_delay']
                            })
                        }
                    }
                })
        )
        .optional()
        .default([])
})
