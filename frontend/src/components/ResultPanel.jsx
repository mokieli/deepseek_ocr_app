import { motion, AnimatePresence } from 'framer-motion'
import { Copy, Download, Sparkles, Loader2, CheckCircle2, ChevronDown } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import BoundingBoxCanvas from './BoundingBoxCanvas'
import { isHTML, isMarkdown } from '../utils/helpers'

export default function ResultPanel({ result, loading, imagePreview, onCopy, onDownload }) {
  // 检查文本格式
  const textIsHTML = isHTML(result?.text)
  const textIsMarkdown = isMarkdown(result?.text)

  return (
    <div className="glass p-6 rounded-2xl space-y-4 h-full">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-purple-400" />
          <h3 className="font-semibold text-gray-200">Results</h3>
        </div>
        
        {result && (
          <div className="flex gap-2">
            <motion.button
              onClick={onCopy}
              className="glass glass-hover p-2 rounded-lg"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              title="Copy to clipboard"
            >
              <Copy className="w-4 h-4" />
            </motion.button>
            <motion.button
              onClick={onDownload}
              className="glass glass-hover p-2 rounded-lg"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              title="Download"
            >
              <Download className="w-4 h-4" />
            </motion.button>
          </div>
        )}
      </div>

      <AnimatePresence mode="wait">
        {loading ? (
          <motion.div
            key="loading"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="flex flex-col items-center justify-center py-20 space-y-4"
          >
            <div className="relative">
              <motion.div
                animate={{ rotate: 360 }}
                transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                className="w-16 h-16 border-4 border-purple-500/20 border-t-purple-500 rounded-full"
              />
              <Loader2 className="w-8 h-8 absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 text-purple-400" />
            </div>
            <p className="text-sm text-gray-400 animate-pulse">
              Processing your image with AI magic...
            </p>
          </motion.div>
        ) : result ? (
          <motion.div
            key="result"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            className="space-y-4"
          >
            {/* Preview with boxes */}
            {imagePreview && result.boxes && result.boxes.length > 0 && (
              <BoundingBoxCanvas 
                imagePreview={imagePreview}
                boxes={result.boxes}
                imageDims={result.image_dims}
              />
            )}

            {/* Text result */}
            <div className="bg-white/5 border border-white/10 rounded-xl p-4 max-h-96 overflow-y-auto">
              {textIsHTML ? (
                <div 
                  className="prose prose-invert prose-sm max-w-none"
                  dangerouslySetInnerHTML={{ __html: result.text }}
                  style={{
                    color: '#e5e7eb',
                  }}
                />
              ) : textIsMarkdown ? (
                <div className="prose prose-invert prose-sm max-w-none">
                  <ReactMarkdown>{result.text}</ReactMarkdown>
                </div>
              ) : (
                <pre className="text-sm text-gray-200 whitespace-pre-wrap font-mono">
                  {result.text}
                </pre>
              )}
            </div>

            {/* Raw Response Viewer */}
            {result.raw_text && (
              <details className="glass rounded-xl overflow-hidden">
                <summary className="px-4 py-3 cursor-pointer flex items-center justify-between hover:bg-white/5 transition-colors">
                  <span className="text-sm font-medium text-gray-300">🔍 Raw Model Response</span>
                  <ChevronDown className="w-4 h-4 text-gray-400" />
                </summary>
                <div className="px-4 py-3 border-t border-white/10 space-y-2">
                  <p className="text-xs text-gray-400 mb-2">Unprocessed output from the model (useful for debugging)</p>
                  <div className="bg-black/30 rounded-lg p-3 max-h-64 overflow-y-auto">
                    <pre className="text-xs text-green-400 font-mono whitespace-pre-wrap break-words select-all">
                      {result.raw_text}
                    </pre>
                  </div>
                  <div className="flex gap-2 mt-2">
                    <button
                      onClick={() => navigator.clipboard.writeText(result.raw_text)}
                      className="text-xs px-3 py-1 bg-white/5 hover:bg-white/10 rounded-lg transition-colors"
                    >
                      Copy Raw
                    </button>
                    <span className="text-xs text-gray-500 py-1">
                      {result.raw_text.length} characters
                    </span>
                  </div>
                </div>
              </details>
            )}

            {/* Advanced Settings Dropdown */}
            <details className="glass rounded-xl overflow-hidden">
              <summary className="px-4 py-3 cursor-pointer flex items-center justify-between hover:bg-white/5 transition-colors">
                <span className="text-sm font-medium text-gray-300">⚙️ Metadata & Debug Info</span>
                <ChevronDown className="w-4 h-4 text-gray-400" />
              </summary>
              <div className="px-4 py-3 border-t border-white/10 space-y-3">
                {result.metadata && (
                  <div>
                    <p className="text-xs text-gray-400 mb-2">Processing Metadata</p>
                    <pre className="text-xs text-gray-500 whitespace-pre-wrap">
                      {JSON.stringify(result.metadata, null, 2)}
                    </pre>
                  </div>
                )}
                {result.boxes?.length > 0 && (
                  <div>
                    <p className="text-xs text-gray-400 mb-2">Parsed Bounding Boxes ({result.boxes.length})</p>
                    <div className="bg-black/30 rounded-lg p-2 space-y-1 max-h-32 overflow-y-auto">
                      {result.boxes.map((box, idx) => (
                        <div key={idx} className="text-xs font-mono">
                          <span className="text-cyan-400">Box {idx + 1}:</span>{' '}
                          <span className="text-purple-400">{box.label}</span>{' '}
                          <span className="text-gray-500">
                            [{box.box.map(n => Math.round(n)).join(', ')}]
                          </span>
                        </div>
                      ))}
                    </div>
                    <p className="text-xs text-gray-500 mt-2">
                      Coordinates are scaled from model output (0-999) to image pixels
                    </p>
                  </div>
                )}
              </div>
            </details>

            {/* Success indicator */}
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="flex items-center justify-center gap-2 text-green-400"
            >
              <CheckCircle2 className="w-5 h-5" />
              <span className="text-sm font-medium">Processing complete!</span>
            </motion.div>
          </motion.div>
        ) : (
          <motion.div
            key="empty"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="flex flex-col items-center justify-center py-20 space-y-4"
          >
            <div className="relative">
              <motion.div
                animate={{ 
                  scale: [1, 1.2, 1],
                  opacity: [0.5, 0.8, 0.5]
                }}
                transition={{ duration: 3, repeat: Infinity }}
                className="w-20 h-20 bg-purple-500/20 rounded-full blur-xl"
              />
              <Sparkles className="w-10 h-10 absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 text-purple-400" />
            </div>
            <div className="text-center">
              <p className="text-lg font-medium text-gray-300">
                Ready to process
              </p>
              <p className="text-sm text-gray-500 mt-1">
                Upload an image and hit analyze to see the magic!
              </p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
