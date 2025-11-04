import ImageOcrPanel from './components/ImageOcrPanel'
import PdfTaskPanel from './components/PdfTaskPanel'
import TaskLookupPanel from './components/TaskLookupPanel'

const App = () => {
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
            即时识别图片，异步处理 PDF，并通过任务 ID 随时跟踪进度。控制台会每秒自动同步最新状态并在任务完成时提供结果压缩包。
          </p>
        </header>

        <div className="grid gap-6 lg:grid-cols-2">
          <ImageOcrPanel />
          <PdfTaskPanel />
        </div>

        <TaskLookupPanel />
      </div>
    </div>
  )
}

export default App
