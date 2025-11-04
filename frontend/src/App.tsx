import { useCallback, useEffect, useId, useMemo, useState, type FormEvent } from 'react'
import {
  API_BASE_URL,
  ImageOCRResponse,
  TaskStatus,
  TaskStatusResponse,
  ocrClient,
} from './api/client'
import {
  AlertCircle,
  CheckCircle2,
  Download,
  FileText,
  ImageIcon,
  Loader2,
  RefreshCcw,
  Upload,
} from 'lucide-react'

const isProcessing = (status?: TaskStatusResponse | null) =>
  status ? status.status === 'pending' || status.status === 'running' : false

const buildDownloadUrl = (path?: string) => {
  if (!path) return undefined
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  try {
    return new URL(path, API_BASE_URL).toString()
  } catch (error) {
    console.warn('Failed to build download URL', error)
    return path
  }
}

const statusBadgeStyles: Record<TaskStatus, string> = {
  pending: 'bg-amber-50 text-amber-700 border border-amber-200',
  running: 'bg-blue-50 text-blue-700 border border-blue-200',
  succeeded: 'bg-emerald-50 text-emerald-700 border border-emerald-200',
  failed: 'bg-rose-50 text-rose-700 border border-rose-200',
}

function App() {
  const [imageResult, setImageResult] = useState<ImageOCRResponse | null>(null)
  const [imageError, setImageError] = useState<string | null>(null)
  const [isImageLoading, setIsImageLoading] = useState(false)
  const [imageFileName, setImageFileName] = useState<string | null>(null)
  const [imagePreviewUrl, setImagePreviewUrl] = useState<string | null>(null)
  const imageInputId = useId()

  const [pdfTaskId, setPdfTaskId] = useState<string | null>(null)
  const [pdfStatus, setPdfStatus] = useState<TaskStatusResponse | null>(null)
  const [pdfError, setPdfError] = useState<string | null>(null)
  const [isPdfUploading, setIsPdfUploading] = useState(false)
  const [pdfFileName, setPdfFileName] = useState<string | null>(null)
  const pdfInputId = useId()

  useEffect(() => {
    return () => {
      if (imagePreviewUrl) {
        URL.revokeObjectURL(imagePreviewUrl)
      }
    }
  }, [imagePreviewUrl])

  const handleImageSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const form = event.currentTarget
    const fileInput = form.elements.namedItem('image') as HTMLInputElement | null
    const file = fileInput?.files?.[0]

    if (!file) {
      setImageError('请选择一张图片')
      return
    }

    setIsImageLoading(true)
    setImageError(null)
    setImageResult(null)
    setImageFileName(file.name)

    try {
      const result = await ocrClient.ocrImage(file)
      setImageResult(result)
    } catch (error) {
      setImageResult(null)
      setImageError((error as Error).message)
    } finally {
      setIsImageLoading(false)
      form.reset()
    }
  }

  const handlePdfSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const form = event.currentTarget
    const fileInput = form.elements.namedItem('pdf') as HTMLInputElement | null
    const file = fileInput?.files?.[0]

    if (!file) {
      setPdfError('请选择一个 PDF 文件')
      return
    }

    setIsPdfUploading(true)
    setPdfError(null)
    setPdfFileName(file.name)

    try {
      const { task_id } = await ocrClient.enqueuePdf(file)
      setPdfTaskId(task_id)
      try {
        const status = await ocrClient.getTaskStatus(task_id)
        setPdfStatus(status)
      } catch (error) {
        console.warn('Initial status fetch failed, will poll later', error)
        setPdfStatus({
          task_id,
          status: 'pending',
          task_type: 'pdf',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          error_message: null,
          result: null,
        })
      }
    } catch (error) {
      setPdfError((error as Error).message)
      setPdfTaskId(null)
      setPdfStatus(null)
    } finally {
      setIsPdfUploading(false)
      form.reset()
    }
  }

  const pollPdfStatus = useCallback(async () => {
    if (!pdfTaskId) return
    try {
      const status = await ocrClient.getTaskStatus(pdfTaskId)
      setPdfStatus(status)
      setPdfError(null)
    } catch (error) {
      setPdfError((error as Error).message)
    }
  }, [pdfTaskId])

  useEffect(() => {
    if (!pdfTaskId) return
    if (!pdfStatus || isProcessing(pdfStatus)) {
      const timer = setInterval(() => {
        pollPdfStatus().catch((error) => {
          console.error('Polling error:', error)
        })
      }, 5000)
      return () => clearInterval(timer)
    }
  }, [pdfTaskId, pdfStatus, pollPdfStatus])

  const imageBoxes = useMemo(() => imageResult?.boxes ?? [], [imageResult])
  const previewBoxes = useMemo(() => {
    if (!imageResult?.boxes?.length || !imageResult.image_dims) return []
    const { w, h } = imageResult.image_dims
    if (!w || !h) return []

    const clamp = (value: number) => Math.min(100, Math.max(0, value))

    return imageResult.boxes
      .map((box, index) => {
        const [x1, y1, x2, y2] = box.box
        const width = Math.max(x2 - x1, 0)
        const height = Math.max(y2 - y1, 0)
        if (width <= 0 || height <= 0) {
          return null
        }
        return {
          id: `${box.label}-${index}`,
          label: box.label,
          left: clamp((x1 / w) * 100),
          top: clamp((y1 / h) * 100),
          width: clamp((width / w) * 100),
          height: clamp((height / h) * 100),
        }
      })
      .filter((box): box is {
        id: string
        label: string
        left: number
        top: number
        width: number
        height: number
      } => Boolean(box))
  }, [imageResult])

  const pdfProgress = pdfStatus?.progress ?? null
  const pdfProgressPercent = Math.min(100, Math.max(0, pdfProgress?.percent ?? 0))
  const showPdfProgress =
    pdfProgress &&
    (pdfProgress.total > 0 || pdfProgress.percent > 0 || (pdfProgress.message && pdfProgress.message.length > 0))

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-10 sm:px-8">
        <header className="flex flex-col gap-3">
          <div className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-slate-500">
            <span className="h-2 w-2 rounded-full bg-emerald-500" />
            vLLM Direct · DeepSeek OCR 控制台
          </div>
          <h1 className="text-3xl font-semibold text-slate-900 sm:text-4xl">统一的 OCR 工作面板</h1>
          <p className="max-w-2xl text-base text-slate-600">
            针对图片任务提供即时推理，同时将 PDF 文档放入队列运行。上传文件后即可在下方查看状态、获取下载链接以及 Markdown 页面摘要。
          </p>
        </header>

        <div className="grid gap-6 lg:grid-cols-2">
          <section className="flex flex-col gap-6 rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 text-sm font-medium text-slate-500">
                  <ImageIcon className="h-4 w-4" />
                  图片 OCR
                </div>
                <h2 className="mt-1 text-xl font-semibold text-slate-900">同步识别单张图片</h2>
                <p className="mt-2 text-sm text-slate-500">
                  支持 JPG、PNG 等常见格式，识别结果会即时返回，包括原始输出与检测框信息。
                </p>
              </div>
            </div>

            <form
              onSubmit={handleImageSubmit}
              className="flex flex-col gap-4 rounded-2xl border border-dashed border-slate-300 bg-slate-50/80 p-5 transition hover:border-slate-400"
            >
              <label
                className="flex flex-col items-center gap-3 text-center text-sm text-slate-600"
                htmlFor={imageInputId}
              >
                <Upload className="h-6 w-6 text-slate-500" />
                <span className="font-medium">
                  {imageFileName ? `已选择：${imageFileName}` : '点击或拖拽文件到此处'}
                </span>
                <span className="text-xs text-slate-400">最大 100 MB，推荐单页图片</span>
              </label>
              <input
                id={imageInputId}
                className="hidden"
                type="file"
                name="image"
                accept="image/*"
                onChange={(event) => {
                  const file = event.currentTarget.files?.[0]
                  if (file) {
                    setImageFileName(file.name)
                    if (imagePreviewUrl) {
                      URL.revokeObjectURL(imagePreviewUrl)
                    }
                    setImagePreviewUrl(URL.createObjectURL(file))
                    setImageResult(null)
                    setImageError(null)
                  } else {
                    if (imagePreviewUrl) {
                      URL.revokeObjectURL(imagePreviewUrl)
                    }
                    setImageFileName(null)
                    setImagePreviewUrl(null)
                  }
                }}
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span className="text-xs text-slate-400">返回内容包含 Markdown 文本、模型原始输出与检测框。</span>
                <button
                  type="submit"
                  disabled={isImageLoading}
                  className="inline-flex items-center gap-2 rounded-full bg-slate-900 px-5 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-slate-700 disabled:cursor-not-allowed disabled:bg-slate-300"
                >
                  {isImageLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <ImageIcon className="h-4 w-4" />}
                  {isImageLoading ? '识别中…' : '开始识别'}
                </button>
              </div>
            </form>
            {imageError && (
              <div className="flex items-start gap-3 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-sm text-rose-700">
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <p>{imageError}</p>
              </div>
            )}

            {imageResult && (
              <div className="flex flex-col gap-4 rounded-2xl border border-slate-200 bg-slate-50/80 p-5">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-lg font-semibold text-slate-900">识别结果</h3>
                  <div className="inline-flex items-center gap-2 rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    成功
                  </div>
                </div>

                {imagePreviewUrl && imageResult.image_dims && (
                  <div className="space-y-2">
                    <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">识别预览</span>
                    <div
                      className="relative w-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-inner"
                      style={{ aspectRatio: `${imageResult.image_dims.w} / ${imageResult.image_dims.h}` }}
                    >
                      <img
                        src={imagePreviewUrl}
                        alt="上传图片预览"
                        className="h-full w-full object-contain"
                      />
                      <div className="pointer-events-none absolute inset-0">
                        {previewBoxes.map((box) => (
                          <div
                            key={box.id}
                            className="absolute rounded border-2 border-emerald-500 bg-emerald-400/15"
                            style={{
                              left: `${box.left}%`,
                              top: `${box.top}%`,
                              width: `${box.width}%`,
                              height: `${box.height}%`,
                            }}
                          >
                            <span className="absolute left-0 top-0 -translate-y-full rounded bg-emerald-500 px-1 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-white">
                              {box.label}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                    {previewBoxes.length === 0 && (
                      <p className="text-xs text-slate-400">模型未返回检测框。</p>
                    )}
                  </div>
                )}

                <div className="space-y-2">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">文本内容</span>
                  <pre className="max-h-80 overflow-auto rounded-2xl bg-white/80 p-4 text-sm leading-relaxed text-slate-800 shadow-inner">
                    {imageResult.text}
                  </pre>
                </div>

                <details className="group rounded-2xl bg-white/70 p-4 shadow-inner">
                  <summary className="flex cursor-pointer items-center gap-2 text-sm font-semibold text-slate-600 transition group-open:text-slate-900">
                    <FileText className="h-4 w-4" />
                    查看原始模型输出
                  </summary>
                  <pre className="mt-3 max-h-72 overflow-auto rounded-xl bg-slate-900/90 p-4 text-xs leading-relaxed text-slate-100">
                    {imageResult.raw_text}
                  </pre>
                </details>

                {imageBoxes.length > 0 && (
                  <div className="rounded-2xl bg-white/80 p-4 shadow-inner">
                    <h4 className="text-sm font-semibold text-slate-700">检测框</h4>
                    <ul className="mt-3 space-y-2 text-sm text-slate-600">
                      {imageBoxes.map((box, index) => (
                        <li key={`${box.label}-${index}`} className="flex items-start justify-between gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2">
                          <span className="font-medium text-slate-700">{box.label}</span>
                          <span className="text-xs text-slate-500">[{box.box.join(', ')}]</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </section>

          <section className="flex flex-col gap-6 rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 text-sm font-medium text-slate-500">
                  <FileText className="h-4 w-4" />
                  PDF OCR
                </div>
                <h2 className="mt-1 text-xl font-semibold text-slate-900">异步队列 · PDF 批量处理</h2>
                <p className="mt-2 text-sm text-slate-500">
                  将 PDF 文档加入队列，系统会在 GPU 上异步推理，稍后通过任务 ID 查询状态与产物。
                </p>
              </div>
            </div>

            <form
              onSubmit={handlePdfSubmit}
              className="flex flex-col gap-4 rounded-2xl border border-dashed border-slate-300 bg-slate-50/80 p-5 transition hover:border-slate-400"
            >
              <label
                className="flex flex-col items-center gap-3 text-center text-sm text-slate-600"
                htmlFor={pdfInputId}
              >
                <Upload className="h-6 w-6 text-slate-500" />
                <span className="font-medium">
                  {pdfFileName ? `已选择：${pdfFileName}` : '上传 PDF 文档以开启任务'}
                </span>
                <span className="text-xs text-slate-400">支持多页文档，后台会拆分、抽取图片与 Markdown</span>
              </label>
              <input
                id={pdfInputId}
                className="hidden"
                type="file"
                name="pdf"
                accept="application/pdf"
                onChange={(event) => {
                  const file = event.currentTarget.files?.[0]
                  if (file) {
                    setPdfFileName(file.name)
                  } else {
                    setPdfFileName(null)
                  }
                }}
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span className="text-xs text-slate-400">任务 ID 可用于状态轮询与下载结果。</span>
                <button
                  type="submit"
                  disabled={isPdfUploading}
                  className="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-5 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:bg-indigo-300"
                >
                  {isPdfUploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileText className="h-4 w-4" />}
                  {isPdfUploading ? '上传中…' : '提交到队列'}
                </button>
              </div>
            </form>
            {pdfError && (
              <div className="flex items-start gap-3 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-sm text-rose-700">
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <p>{pdfError}</p>
              </div>
            )}

            {pdfTaskId && (
              <div className="flex flex-col gap-4 rounded-2xl border border-slate-200 bg-slate-50/80 p-5">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h3 className="text-lg font-semibold text-slate-900">任务状态</h3>
                    <p className="mt-1 text-xs font-medium text-slate-500">ID：{pdfTaskId}</p>
                  </div>
                  {pdfStatus && (
                    <span className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold ${statusBadgeStyles[pdfStatus.status]}`}>
                      {pdfStatus.status === 'failed' ? (
                        <AlertCircle className="h-3.5 w-3.5" />
                      ) : pdfStatus.status === 'succeeded' ? (
                        <CheckCircle2 className="h-3.5 w-3.5" />
                      ) : (
                        <Loader2 className="h-3.5 w-3.5 animate-spin text-inherit" />
                      )}
                      {pdfStatus.status === 'pending' && '排队中'}
                      {pdfStatus.status === 'running' && '执行中'}
                      {pdfStatus.status === 'succeeded' && '已完成'}
                      {pdfStatus.status === 'failed' && '失败'}
                    </span>
                  )}
                </div>

                {pdfStatus?.error_message && (
                  <div className="flex items-start gap-2 rounded-xl border border-rose-200 bg-white p-3 text-xs text-rose-700">
                    <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                    <span>{pdfStatus.error_message}</span>
                  </div>
                )}

                {showPdfProgress && pdfProgress && (
                  <div className="space-y-2 rounded-2xl border border-slate-200 bg-white/80 p-4 shadow-inner">
                    <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
                      <span>处理进度</span>
                      <span>{Math.round(pdfProgressPercent)}%</span>
                    </div>
                    <div className="h-2 w-full rounded-full bg-slate-200">
                      <div
                        className="h-full rounded-full bg-indigo-500 transition-all duration-500"
                        style={{ width: `${pdfProgressPercent}%` }}
                      />
                    </div>
                    {pdfProgress.message && (
                      <p className="text-xs text-slate-500">{pdfProgress.message}</p>
                    )}
                  </div>
                )}

                <div className="flex flex-wrap items-center gap-3">
                  <button
                    type="button"
                    onClick={() => {
                      void pollPdfStatus()
                    }}
                    className="inline-flex items-center gap-2 rounded-full border border-slate-300 bg-white px-4 py-2 text-xs font-medium text-slate-600 transition hover:border-slate-400 hover:text-slate-900"
                  >
                    <RefreshCcw className="h-3.5 w-3.5" />
                    立即刷新
                  </button>
                  <p className="text-xs text-slate-400">
                    正常情况下会每 5 秒自动轮询，支持手动刷新。
                  </p>
                </div>

                {pdfStatus?.result && (
                  <div className="space-y-4">
                    <div>
                      <h4 className="text-sm font-semibold text-slate-700">下载链接</h4>
                      <ul className="mt-3 space-y-2 text-sm text-slate-600">
                        {pdfStatus.result.markdown_url && (
                          <li>
                            <a
                              className="inline-flex items-center gap-2 rounded-full border border-slate-300 px-4 py-2 text-xs font-medium text-indigo-600 transition hover:border-indigo-400 hover:text-indigo-700"
                              href={buildDownloadUrl(pdfStatus.result.markdown_url)}
                              target="_blank"
                              rel="noreferrer"
                            >
                              <Download className="h-3.5 w-3.5" />
                              Markdown 摘要
                            </a>
                          </li>
                        )}
                        {pdfStatus.result.archive_url && (
                          <li>
                            <a
                              className="inline-flex items-center gap-2 rounded-full border border-slate-300 px-4 py-2 text-xs font-medium text-indigo-600 transition hover:border-indigo-400 hover:text-indigo-700"
                              href={buildDownloadUrl(pdfStatus.result.archive_url)}
                              target="_blank"
                              rel="noreferrer"
                            >
                              <Download className="h-3.5 w-3.5" />
                              打包 ZIP
                            </a>
                          </li>
                        )}
                        {pdfStatus.result.raw_json_url && (
                          <li>
                            <a
                              className="inline-flex items-center gap-2 rounded-full border border-slate-300 px-4 py-2 text-xs font-medium text-indigo-600 transition hover:border-indigo-400 hover:text-indigo-700"
                              href={buildDownloadUrl(pdfStatus.result.raw_json_url)}
                              target="_blank"
                              rel="noreferrer"
                            >
                              <Download className="h-3.5 w-3.5" />
                              原始 JSON
                            </a>
                          </li>
                        )}
                        {pdfStatus.result.image_urls.map((url) => (
                          <li key={url}>
                            <a
                              className="inline-flex items-center gap-2 rounded-full border border-slate-300 px-4 py-2 text-xs font-medium text-indigo-600 transition hover:border-indigo-400 hover:text-indigo-700"
                              href={buildDownloadUrl(url)}
                              target="_blank"
                              rel="noreferrer"
                            >
                              <Download className="h-3.5 w-3.5" />
                              {url.split('/').pop()}
                            </a>
                          </li>
                        ))}
                      </ul>
                    </div>

                    {pdfStatus.result.pages?.length ? (
                      <div className="space-y-3">
                        <h4 className="text-sm font-semibold text-slate-700">页面摘要</h4>
                        <div className="space-y-2">
                          {pdfStatus.result.pages.map((page) => (
                            <details
                              key={page.index}
                              className="group rounded-2xl border border-slate-200 bg-white/80 p-4 shadow-inner"
                            >
                              <summary className="flex cursor-pointer items-center justify-between gap-2 text-sm font-semibold text-slate-600 transition group-open:text-slate-900">
                                <span>第 {page.index + 1} 页</span>
                                <span className="text-xs font-medium text-slate-400 group-open:text-indigo-500">
                                  点击展开
                                </span>
                              </summary>
                              <pre className="mt-3 max-h-64 overflow-auto rounded-xl bg-slate-900/90 p-4 text-xs leading-relaxed text-slate-100">
                                {page.markdown}
                              </pre>
                            </details>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </div>
                )}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}

export default App
