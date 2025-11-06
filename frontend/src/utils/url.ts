import { API_BASE_URL } from '../api/client'

export const buildDownloadUrl = (path?: string) => {
  if (!path) return undefined
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  try {
    return new URL(path, API_BASE_URL).toString()
  } catch (error: unknown) {
    console.warn('Failed to build download URL', error)
    return path
  }
}
