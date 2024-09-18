import * as z from 'zod';
import { isGoDuration } from '@/utils/validation';

export const smtpConfigSchema = z.object({
    enabled: z.boolean().describe('Enabled status').default(false),
    username: z.string().describe('SMTP username').email(),
    host: z.string().describe('SMTP host').default('smtp.gmail.com'),
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
        .describe('SMTP port')
        .default(587),
    password: z.string().describe('SMTP password').default(''),
    max_conns: z
        .number({
            invalid_type_error: 'Must be a number.'
        })
        .min(1, {
            message: 'Maximum connections must be at least 1.'
        })
        .describe('Maximum concurrent connections')
        .default(2),
    idle_timeout: z
        .string()
        .describe('Idle timeout duration')
        .refine(isGoDuration, {
            message: 'Invalid duration format. Should be a number followed by s (seconds), m (minutes), or h (hours).'
        })
        .default('15s'),
    wait_timeout: z
        .string()
        .describe('Wait timeout duration')
        .refine(isGoDuration, {
            message: 'Invalid duration format. Should be a number followed by s (seconds), m (minutes), or h (hours).'
        })
        .default('5s'),
    auth_protocol: z
        .enum(['plain', 'login', 'cram', 'none'])
        .describe('Authentication protocol')
        .default('plain'),
    email_address: z.string().describe('Email address').email(),
    max_msg_retries: z
        .number({
            invalid_type_error: 'Must be a number.'
        })
        .min(0, {
            message: 'Max message retries must be at least 0.'
        })
        .max(100, {
            message: 'Max message retries must be at most 100.'
        })
        .describe('Maximum message retries')
        .default(2)
});
