import { format } from 'date-fns'

export function formatTime (t) {
    try {
        return format(t, 'h:mm a')
    } catch (error) {
        console.error("error parsing time", error)
        return ''
    }
}