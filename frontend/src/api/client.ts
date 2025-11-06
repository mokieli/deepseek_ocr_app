import axios, { AxiosInstance } from 'axios'
import { isErrorLike } from '../utils/errors'

export interface BoundingBox {
  label: string
  box: number[]
}

export interface TaskTiming {
  queued_at?: string | null
  started_at?: string | null
  finished_at?: string | null
  duration_ms?: number | null
}

export interface ImageOCRResponse {
  success: boolean
  text: string
  raw_text: string
  boxes: BoundingBox[]
  image_dims?: { w: number; h: number }
  task_id?: string | null
  timing?: TaskTiming | null
  duration_ms?: number | null
}

export interface TaskCreateResponse {
  task_id: string
}

export interface PdfPageResult {
  index: number
  markdown: string
  raw_text: string
  image_assets: string[]
  boxes: BoundingBox[]
}

export interface TaskResult {
  markdown_url?: string
  raw_json_url?: string
  archive_url?: string
  image_urls: string[]
  pages: PdfPageResult[]
}

export type TaskStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export interface TaskProgress {
  current: number
  total: number
  percent: number
  message?: string | null
  pages_completed?: number | null
  pages_total?: number | null
}

export interface TaskStatusResponse {
  task_id: string
  status: TaskStatus
  task_type: 'pdf' | 'image'
  created_at: string
  updated_at: string
  error_message?: string | null
  result?: TaskResult | null
  progress?: TaskProgress | null
  timing?: TaskTiming | null
}

export const API_BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8001'

class OCRClient {
  private client: AxiosInstance

  constructor(baseURL: string = API_BASE_URL) {
    this.client = axios.create({
      baseURL,
      timeout: 300000,
    })

    this.client.interceptors.response.use(
      (response) => response,
      (error: unknown) => {
        // Derive a robust error message without using any
        let message = 'Request failed'
        if (error && typeof error === 'object') {
          const maybeResponse = (error as Record<string, unknown>).response
          if (maybeResponse && typeof maybeResponse === 'object') {
            const data = (maybeResponse as Record<string, unknown>).data
            if (data && typeof data === 'object') {
              const detail = (data as Record<string, unknown>).detail
              if (typeof detail === 'string') {
                message = detail
              }
            }
          }
        }
        if (message === 'Request failed' && isErrorLike(error)) {
          message = error.message
        }
        throw new Error(message)
      }
    )
  }

  async healthCheck(): Promise<unknown> {
    const { data } = await this.client.get<unknown>('/health')
    return data
  }

  async ocrImage(file: File): Promise<ImageOCRResponse> {
    const formData = new FormData()
    formData.append('image', file)

    const { data } = await this.client.post<ImageOCRResponse>('/api/ocr/image', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
  }

  async enqueuePdf(file: File): Promise<TaskCreateResponse> {
    const formData = new FormData()
    formData.append('pdf', file)

    const { data } = await this.client.post<TaskCreateResponse>('/api/ocr/pdf', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
  }

  async getTaskStatus(taskId: string): Promise<TaskStatusResponse> {
    const { data } = await this.client.get<TaskStatusResponse>(`/api/tasks/${taskId}`)
    return data
  }
}

export const ocrClient = new OCRClient()

export default OCRClient
