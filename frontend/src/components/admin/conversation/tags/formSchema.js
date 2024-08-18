import * as z from 'zod';

export const formSchema = z.object({
    name: z
        .string({
            required_error: 'Tag name is required.'
        })
        .min(1, {
            message: 'First name must be at least 1 character.'
        }),

});
