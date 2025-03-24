import * as z from 'zod'

const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/

export const formSchema = z.object({
    name: z.string().min(1, 'Name is required.'),
    description: z.string().min(1, 'Description is required.'),
    is_always_open: z.boolean().default(true),
    hours: z.record(
        z.object({
            open: z.string().regex(timeRegex, 'Invalid time format (HH:mm)'),
            close: z.string().regex(timeRegex, 'Invalid time format (HH:mm)')
        })
    ).optional()
}).superRefine((data, ctx) => {
    if (data.is_always_open === false) {
        if (!data.hours || Object.keys(data.hours).length === 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: 'Business hours are required',
                path: ['hours']
            })
        } else {
            for (const day in data.hours) {
                if (!data.hours[day].open || !data.hours[day].close) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        message: 'Open and close times are required for each day.',
                        path: ['hours', day]
                    })
                }
            }
        }
    }
})
