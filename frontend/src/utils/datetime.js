import { format, differenceInMinutes, differenceInHours, differenceInDays } from 'date-fns'

export function getRelativeTime (timestamp, now = new Date()) {
  try {
    const mins = differenceInMinutes(now, timestamp)
    const hours = differenceInHours(now, timestamp)
    const days = differenceInDays(now, timestamp)

    if (mins === 0) return 'Just now'
    if (mins < 60) return `${mins} mins ago`
    if (hours < 24) return `${hours} hrs ago`
    if (days < 7) return `${days} days ago`
    return format(timestamp, 'MMMM d, yyyy h:mm a')
  } catch (error) {
    console.error('Error parsing time', error, 'timestamp', timestamp)
    return ''
  }
}