import * as z from 'zod';

export const userFormSchema = z.object({
  first_name: z
    .string({
      required_error: 'First name is required.'
    })
    .min(2, {
      message: 'First name must be at least 2 characters.'
    }),

  last_name: z.string().optional(),

  email: z
    .string({
      required_error: 'Email is required.'
    })
    .email({
      message: 'Invalid email address.'
    }),

  teams: z.array(z.string()).optional(),

  roles: z.array(z.string()).optional(),

  send_welcome_email: z.boolean().optional(),
});
