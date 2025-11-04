import type { TaskStatus, TaskStatusResponse } from '../api/client'

export const statusBadgeStyles: Record<TaskStatus, string> = {
  pending: 'bg-amber-50 text-amber-700 border border-amber-200',
  running: 'bg-blue-50 text-blue-700 border border-blue-200',
  succeeded: 'bg-emerald-50 text-emerald-700 border border-emerald-200',
  failed: 'bg-rose-50 text-rose-700 border border-rose-200',
}

export const isProcessing = (status?: TaskStatusResponse | null) =>
  status ? status.status === 'pending' || status.status === 'running' : false
