import * as z from 'zod';

export const formSchema = z
    .object({
        name: z.string({
            required_error: 'Rule name is required.',
        }),
        description: z.string({
            required_error: 'Rule description is required.',
        }),
        type: z.string({
            required_error: 'Rule type is required.',
        }),
        events: z.array(z.string()).optional(),
    })
    .superRefine((data, ctx) => {
        if (data.type === 'conversation_update' && (!data.events || data.events.length === 0)) {
            ctx.addIssue({
                path: ['events'],
                message: 'Please select at least one event.',
                code: z.ZodIssueCode.custom,
            });
        }
    });
