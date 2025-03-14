import * as z from 'zod'
import { isGoDuration } from '@/utils/strings'

export const formSchema = z.object({
  name: z.string().min(1, 'Required'),
  from: z.string().min(1, 'Required'),
  enabled: z.boolean().optional(),
  csat_enabled: z.boolean().optional(),
  imap: z.object({
    host: z.string().min(1, 'Required'),
    port: z.number().min(1).max(65535),
    mailbox: z.string().min(1, 'Required'),
    username: z.string().min(1, 'Required'),
    password: z.string().min(1, 'Required'),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    scan_inbox_since: z.string().min(1, 'Required').refine(isGoDuration, {
      message: 'Invalid duration. Please use a valid duration format (e.g. 1h, 30m, 1h30m, 48h, etc.)'
    }),
    read_interval: z.string().min(1, 'Required').refine(isGoDuration)
  }),

  smtp: z.object({
    host: z.string().min(1, 'Required'),
    port: z.number().min(1).max(65535),
    username: z.string().min(1, 'Required'),
    password: z.string().min(1, 'Required'),
    max_conns: z.number().min(1),
    max_msg_retries: z.number().min(0).max(100),
    idle_timeout: z.string().min(1, 'Required').refine(isGoDuration),
    wait_timeout: z.string().min(1, 'Required').refine(isGoDuration),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    hello_hostname: z.string().optional(),
    auth_protocol: z.enum(['login', 'cram', 'plain', 'none'])
  })
})
