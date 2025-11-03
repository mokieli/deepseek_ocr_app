import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Sparkles, Zap, Settings } from 'lucide-react'
import ImageUpload from './components/ImageUpload'
import ModeSelector from './components/ModeSelector'
import ResultPanel from './components/ResultPanel'
import AdvancedSettings from './components/AdvancedSettings'
import useOCR from './hooks/useOCR'
import { downloadTextFile, getFileExtension } from './utils/helpers'

function App() {
  const [mode, setMode] = useState('plain_ocr')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [includeCaption, setIncludeCaption] = useState(false)
  
  // Form state
  const [prompt, setPrompt] = useState('')
  const [findTerm, setFindTerm] = useState('')
  const [advancedSettings, setAdvancedSettings] = useState({
    base_size: 1024,
    image_size: 640,
    crop_mode: true,
    test_compress: false
  })

  // 使用 OCR Hook
  const [state, actions] = useOCR()
  const { imagePreview, result, loading, error } = state
  const { handleImageSelect, handleSubmit } = actions

  const onSubmit = async () => {
    await handleSubmit({
      mode,
      prompt,
      grounding: mode === 'find_ref',
      includeCaption,
      findTerm,
      schema: '',
      baseSize: advancedSettings.base_size,
      imageSize: advancedSettings.image_size,
      cropMode: advancedSettings.crop_mode,
      testCompress: advancedSettings.test_compress,
    })
  }

  const handleCopy = () => {
    if (result?.text) {
      navigator.clipboard.writeText(result.text)
    }
  }

  const handleDownload = () => {
    if (!result?.text) return
    const ext = getFileExtension(mode)
    downloadTextFile(result.text, `deepseek-ocr-result.${ext}`)
  }

  return (
    <div className="min-h-screen relative overflow-hidden">
      {/* Animated background */}
      <div className="fixed inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-purple-900/20 via-pink-900/20 to-cyan-900/20" />
        <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHZpZXdCb3g9IjAgMCA2MCA2MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZyBmaWxsPSJub25lIiBmaWxsLXJ1bGU9ImV2ZW5vZGQiPjxwYXRoIGQ9Ik0zNiAxOGMzLjMxIDAgNiAyLjY5IDYgNnMtMi42OSA2LTYgNi02LTIuNjktNi02IDIuNjktNiA2LTZ6TTI0IDZjMy4zMSAwIDYgMi42OSA2IDZzLTIuNjkgNi02IDYtNi0yLjY5LTYtNiAyLjY5LTYgNi02ek00OCAzNmMzLjMxIDAgNiAyLjY5IDYgNnMtMi42OSA2LTYgNi02LTIuNjktNi02IDIuNjktNiA2LTZ6IiBzdHJva2U9InJnYmEoMTQ3LCA1MSwgMjM0LCAwLjEpIiBzdHJva2Utd2lkdGg9IjIiLz48L2c+PC9zdmc+')] opacity-30" />
        <motion.div
          className="absolute top-20 left-20 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl"
          animate={{
            scale: [1, 1.2, 1],
            opacity: [0.3, 0.5, 0.3],
          }}
          transition={{
            duration: 8,
            repeat: Infinity,
            ease: "easeInOut"
          }}
        />
        <motion.div
          className="absolute bottom-20 right-20 w-96 h-96 bg-cyan-500/10 rounded-full blur-3xl"
          animate={{
            scale: [1.2, 1, 1.2],
            opacity: [0.5, 0.3, 0.5],
          }}
          transition={{
            duration: 8,
            repeat: Infinity,
            ease: "easeInOut"
          }}
        />
      </div>

      {/* Header */}
      <header className="sticky top-0 z-50 glass border-b border-white/10">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <motion.div 
              className="flex items-center gap-3"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
            >
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-r from-purple-500 to-cyan-500 rounded-xl blur-lg opacity-75" />
                <div className="relative bg-gradient-to-br from-purple-600 to-cyan-500 p-2 rounded-xl">
                  <Sparkles className="w-6 h-6" />
                </div>
              </div>
              <div>
                <h1 className="text-2xl font-bold gradient-text">DeepSeek OCR</h1>
                <p className="text-xs text-gray-400">Next-Gen Vision AI</p>
              </div>
            </motion.div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid lg:grid-cols-2 gap-6">
          {/* Left Panel - Upload & Controls */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="space-y-6"
          >
            {/* Mode Selector with integrated inputs */}
            <ModeSelector 
              mode={mode} 
              onModeChange={setMode}
              prompt={prompt}
              onPromptChange={setPrompt}
              findTerm={findTerm}
              onFindTermChange={setFindTerm}
            />

            {/* Image Upload */}
            <ImageUpload 
              onImageSelect={handleImageSelect}
              preview={imagePreview}
            />

            {/* Advanced Settings Toggle */}
            <motion.button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="w-full glass px-4 py-3 rounded-2xl flex items-center justify-between hover:bg-white/5 transition-colors"
              whileHover={{ scale: 1.01 }}
              whileTap={{ scale: 0.99 }}
            >
              <div className="flex items-center gap-2">
                <Settings className="w-4 h-4 text-purple-400" />
                <span className="text-sm font-medium text-gray-300">Advanced Settings</span>
              </div>
              <motion.div
                animate={{ rotate: showAdvanced ? 180 : 0 }}
                transition={{ duration: 0.3 }}
              >
                <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </motion.div>
            </motion.button>

            {/* Advanced Settings Panel */}
            <AnimatePresence>
              {showAdvanced && (
                <AdvancedSettings
                  settings={advancedSettings}
                  onSettingsChange={setAdvancedSettings}
                  includeCaption={includeCaption}
                  onIncludeCaptionChange={setIncludeCaption}
                />
              )}
            </AnimatePresence>

            {/* Action Button */}
            <motion.button
              onClick={onSubmit}
              disabled={!imagePreview || loading}
              className={`w-full relative overflow-hidden rounded-2xl p-[2px] ${
                !imagePreview || loading ? 'opacity-50 cursor-not-allowed' : ''
              }`}
              whileHover={!loading && imagePreview ? { scale: 1.02 } : {}}
              whileTap={!loading && imagePreview ? { scale: 0.98 } : {}}
            >
              <div className="absolute inset-0 bg-gradient-to-r from-purple-600 via-pink-600 to-cyan-600 animate-gradient" />
              <div className="relative bg-dark-100 px-8 py-4 rounded-2xl flex items-center justify-center gap-3">
                {loading ? (
                  <>
                    <motion.div
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                      className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full"
                    />
                    <span className="font-semibold">Processing Magic...</span>
                  </>
                ) : (
                  <>
                    <Zap className="w-5 h-5" />
                    <span className="font-semibold">Analyze Image</span>
                  </>
                )}
              </div>
            </motion.button>

            {error && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="glass p-4 rounded-2xl border-red-500/50 bg-red-500/10"
              >
                <p className="text-sm text-red-400">{error}</p>
              </motion.div>
            )}
          </motion.div>

          {/* Right Panel - Results */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <ResultPanel 
              result={result}
              loading={loading}
              imagePreview={imagePreview}
              onCopy={handleCopy}
              onDownload={handleDownload}
            />
          </motion.div>
        </div>
      </main>

      {/* Footer */}
      <footer className="mt-20 border-t border-white/10 glass">
        <div className="max-w-7xl mx-auto px-6 py-8 text-center">
          <p className="text-sm text-gray-400">
            Powered by <span className="gradient-text font-semibold">DeepSeek-OCR</span> • 
            Built with <span className="text-pink-400">♥</span> using React + FastAPI
          </p>
        </div>
      </footer>
    </div>
  )
}

export default App
