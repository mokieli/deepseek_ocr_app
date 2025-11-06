export interface ErrorLike {
  message: string
}

export const isErrorLike = (e: unknown): e is ErrorLike => {
  if (typeof e !== 'object' || e === null) return false
  return 'message' in e && typeof (e as Record<string, unknown>).message === 'string'
}

export const getErrorMessage = (e: unknown): string => {
  if (typeof e === 'string') return e
  if (isErrorLike(e)) return e.message
  try {
    return JSON.stringify(e)
  } catch {
    return 'Unknown error'
  }
}

