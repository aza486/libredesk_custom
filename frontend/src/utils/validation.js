// Function to validate Go-like duration
export const isGoDuration = (value) => {
  const regex = /^[0-9]+[smh]$/
  return regex.test(value)
}
