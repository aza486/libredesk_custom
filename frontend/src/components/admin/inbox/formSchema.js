import * as z from 'zod'
import { isGoDuration } from '@/utils/strings'

export const formSchema = z.object({
  name: z.string().describe('Name').default(''),
  from: z.string().describe('From address').default(''),
  csat_enabled: z.boolean().describe('Enable CSAT').optional(),
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
        .describe('Port')
        .default(993),
      mailbox: z.string().describe('Mailbox name').default('INBOX'),
      username: z.string().describe('Username'),
      password: z.string().describe('Password'),
      read_interval: z
        .string()
        .describe('Email scan interval')
        .refine(isGoDuration, {
          message:
            'Invalid duration format. Should be a number followed by s (seconds), m (minutes), or h (hours).'
        })
        .default('120s')
    })
    .describe('IMAP client')
    .default({
      host: 'imap.gmail.com',
      port: 993,
      mailbox: 'INBOX',
      username: '',
      password: '',
      read_interval: '30s'
    }),
  smtp: z
    .object({
      host: z.string().describe('Host').default('smtp.google.com'),
      port: z
        .number({ invalid_type_error: 'Port must be a number.' })
        .min(1, { message: 'Port must be at least 1.' })
        .max(65535, { message: 'Port must be at most 65535.' })
        .describe('Port')
        .default(587),
      username: z.string().describe('Username'),
      password: z.string().describe('Password'),
      max_conns: z
        .number({ invalid_type_error: 'Must be a number.' })
        .min(1, { message: 'Must be at least 1.' })
        .describe('Maximum concurrent connections to the server.')
        .default(2),
      max_msg_retries: z
        .number({ invalid_type_error: 'Must be a number.' })
        .min(0, { message: 'Must be at least 0.' })
        .max(100, { message: 'Max retries allowed are 100.' })
        .describe('Number of times to retry when a message fails.')
        .default(2),
      idle_timeout: z
        .string()
        .describe(
          'Time to wait for new activity on a connection before closing it and removing it from the pool (s for seconds, m for minutes, h for hours).'
        )
        .refine(isGoDuration, {
          message:
            'Invalid duration format. Should be a number followed by s (seconds), m (minutes), or h (hours).'
        })
        .default('5s'),
      wait_timeout: z
        .string()
        .describe(
          'Time to wait for new activity on a connection before closing it and removing it from the pool (s for seconds, m for minutes, h for hours).'
        )
        .refine(isGoDuration, {
          message:
            'Invalid duration format. Should be a number followed by s (seconds), m (minutes), or h (hours).'
        })
        .default('5s'),
      auth_protocol: z.enum(['login', 'cram', 'plain', 'none']).default('plain').optional(),
    })
    .describe('SMTP server')
})
