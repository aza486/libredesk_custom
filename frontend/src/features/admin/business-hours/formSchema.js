import * as z from 'zod'

const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/

export const createFormSchema = (t) => z.object({
    name: z.string().min(1, t('form.error.name.required')),
    description: z.string().min(1, t('form.error.description.required')),
    is_always_open: z.boolean().default(true),
    hours: z.record(
        z.object({
            open: z.string().regex(timeRegex, t('form.error.time.invalid')),
            close: z.string().regex(timeRegex, t('form.error.time.invalid')),
        })
    ).optional()
}).superRefine((data, ctx) => {
    if (data.is_always_open === false) {
        if (!data.hours || Object.keys(data.hours).length === 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: t('admin.business_hours.hours.required'),
                path: ['hours']
            })
        } else {
            for (const day in data.hours) {
                if (!data.hours[day].open || !data.hours[day].close) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        message: t('admin.business_hours.open_close.required'),
                        path: ['hours', day]
                    })
                }
            }
        }
    }
})
