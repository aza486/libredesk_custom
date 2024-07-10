import * as z from 'zod'

export const emailChannelFormSchema = z.object({
  name: z.string().describe('Name').default(''),
  from: z.string().describe('From address').default(''),
  imap: z
    .object({
      host: z.string().describe('Host').default('imap.gmail.com'),
      port: z
        .number({
          invalid_type_error: 'Port must be a number.'
        })
        .min(1, {
          message: 'Port must be at least 1.'
        })
        .max(65535, {
          message: 'Port must be at most 65535.'
        })
        .default(993)
        .describe('Port'),
      mailbox: z.string().describe('Mailbox name').default('INBOX'),
      username: z.string().describe('Username'),
      password: z.string().describe('Password'),
      read_interval: z.string().describe('Email scan interval').default('30s')
    })
    .describe('IMAP client'),
  smtp: z
    .array(
      z
        .object({
          host: z.string().describe('Host').default('smtp.yourmailserver.com'),
          port: z
            .number({
              invalid_type_error: 'Port must be a number.'
            })
            .min(1, {
              message: 'Port must be at least 1.'
            })
            .max(65535, {
              message: 'Port must be at most 65535.'
            })
            .default(25)
            .describe('Port'),
          username: z.string().describe('Username'),
          password: z.string().describe('Password'),
          max_conns: z
            .number({
              invalid_type_error: 'Must be a number.'
            })
            .min(1, {
              message: 'Must be at least 1.'
            })
            .default(10)
            .describe('Maximum concurrent connections to the server.'),
          max_msg_retries: z
            .number({
              invalid_type_error: 'Must be a number.'
            })
            .min(0, {
              message: 'Must be at least 0.'
            })
            .max(100, {
              message: 'Max retries allowed are 100.'
            })
            .default(2)
            .describe('Number of times to retry when a message fails.'),
          idle_timeout: z
            .string()
            .default('5s')
            .describe(
              'Time to wait for new activity on a connection before closing it and removing it from the pool (s for second, m for minute).'
            ),
          wait_timeout: z
            .string()
            .default('5s')
            .describe(
              'Time to wait for new activity on a connection before closing it and removing it from the pool (s for second, m for minute).'
            ),
          auth_protocol: z.enum(['login', 'cram', 'plain', 'none']).default('none').optional()
        })
        .describe('SMTP')
    )
    .describe('SMTP servers')
})
