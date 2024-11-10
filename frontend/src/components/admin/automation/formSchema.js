import * as z from 'zod'

export const formSchema = z.object({
    name: z.string({
        required_error: 'Rule name is required.'
    }),
    description: z.string({
        required_error: 'Rule description is required.'
    }),
    type: z.string({
        required_error: 'Rule type is required.'
    }),
    events: z.array(z.string()).min(1, 'Please select at least one event.'),
})
