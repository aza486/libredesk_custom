import * as z from 'zod'

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

  send_welcome_email: z.boolean().optional(),

  teams: z.array(z.string()).default([]),

  roles: z.array(z.string()).min(1, 'Please select at least one role.'),

  new_password: z
    .string()
    .regex(/^$|^(?=.*[A-Z])(?=.*\d)[A-Za-z\d]{8,50}$/, {
      message: 'Password must be between 8 and 50 characters long, contain at least one uppercase letter and one number.'
    })
    .optional(),
  enabled: z.boolean().optional().default(true)
})
