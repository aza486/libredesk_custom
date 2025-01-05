import * as z from 'zod';

export const formSchema = z
  .object({
    name: z.string({
      required_error: 'Template name is required.',
    }),
    body: z.string({
      required_error: 'Template content is required.',
    }),
    type: z.string().optional(),
    subject: z.string().optional(),
    is_default: z.boolean().optional(),
  })
  .superRefine((data, ctx) => {
    if (data.type !== 'email_outgoing' && !data.subject) {
      ctx.addIssue({
        path: ['subject'],
        message: 'Subject is required.',
        code: z.ZodIssueCode.custom,
      });
    }
  });
