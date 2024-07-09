import { format, differenceInMinutes, differenceInHours, differenceInDays } from 'date-fns';

export function formatTime(t) {
    try {
        const now = new Date();
        const minutesDifference = differenceInMinutes(now, t);
        const hoursDifference = differenceInHours(now, t);
        const daysDifference = differenceInDays(now, t);

        if (minutesDifference === 0) {
            return `Just now`;
        } else if (minutesDifference < 60) {
            return `${minutesDifference} minutes ago`;
        } else if (hoursDifference < 24) {
            return `${hoursDifference} hours ago`;
        } else if (daysDifference < 7) {
            return `${daysDifference} days ago`;
        } else {
            return format(t, 'MMMM d, yyyy h:mm a');
        }
    } catch (error) {
        console.error("error parsing time", error, "time", t)
        return ''
    }
}
