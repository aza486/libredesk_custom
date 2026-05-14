export function formatBytes (bytes) {
    if (bytes < 1024 * 1024) {
        return (bytes / 1024).toFixed(2) + ' KB'
    } else {
        return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
    }
}

export function getThumbFilepath (filepath) {
    const urlParts = filepath.split('/')
    const filename = urlParts.pop()
    return `/uploads/thumb_${filename}`
}

export function downloadUrl (url) {
    if (!url) return url
    const separator = url.includes('?') ? '&' : '?'
    return `${url}${separator}download=1`
}
