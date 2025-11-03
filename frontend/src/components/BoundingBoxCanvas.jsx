/**
 * 边界框 Canvas 组件
 * 在图像上绘制检测到的边界框
 */
import { useEffect, useRef, useCallback } from 'react'
import { getBoxColor } from '../utils/helpers'

export default function BoundingBoxCanvas({ imagePreview, boxes, imageDims }) {
  const canvasRef = useRef(null)
  const imgRef = useRef(null)

  /**
   * 绘制边界框
   */
  const drawBoxes = useCallback(() => {
    if (!boxes?.length || !canvasRef.current || !imgRef.current) {
      console.log('❌ Cannot draw - missing:', {
        hasBoxes: !!boxes?.length,
        hasCanvas: !!canvasRef.current,
        hasImgRef: !!imgRef.current,
      })
      return
    }

    console.log('🎨 Drawing boxes:', boxes)

    const img = imgRef.current
    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')

    console.log('📐 Image dimensions:', {
      displayWidth: img.offsetWidth,
      displayHeight: img.offsetHeight,
      naturalWidth: img.naturalWidth,
      naturalHeight: img.naturalHeight,
      imageDims,
    })

    // 设置 canvas 尺寸匹配显示的图像
    canvas.width = img.offsetWidth
    canvas.height = img.offsetHeight

    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // 计算缩放因子
    const scaleX = img.offsetWidth / (imageDims?.w || img.naturalWidth)
    const scaleY = img.offsetHeight / (imageDims?.h || img.naturalHeight)

    console.log('📏 Scale factors:', { scaleX, scaleY })

    // 绘制每个边界框
    boxes.forEach((box, idx) => {
      const [x1, y1, x2, y2] = box.box
      const color = getBoxColor(idx)

      // 缩放坐标
      const sx = x1 * scaleX
      const sy = y1 * scaleY
      const sw = (x2 - x1) * scaleX
      const sh = (y2 - y1) * scaleY

      console.log(`📦 Box ${idx} (${box.label}):`, {
        original: [x1, y1, x2, y2],
        scaled: [sx, sy, sx + sw, sy + sh],
        dimensions: { width: sw, height: sh },
      })

      // 绘制半透明填充
      ctx.fillStyle = color + '33'
      ctx.fillRect(sx, sy, sw, sh)

      // 绘制霓虹边框
      ctx.strokeStyle = color
      ctx.lineWidth = 4
      ctx.shadowColor = color
      ctx.shadowBlur = 10
      ctx.strokeRect(sx, sy, sw, sh)
      ctx.shadowBlur = 0

      // 标签背景
      if (box.label) {
        ctx.font = 'bold 14px Inter'
        const metrics = ctx.measureText(box.label)
        const padding = 8
        const labelHeight = 24

        ctx.fillStyle = color
        ctx.fillRect(sx, sy - labelHeight, metrics.width + padding * 2, labelHeight)

        // 标签文本
        ctx.fillStyle = '#000'
        ctx.fillText(box.label, sx + padding, sy - 7)
      }
    })

    console.log('✅ Finished drawing', boxes.length, 'boxes')
  }, [boxes, imageDims])

  /**
   * 图像加载后绘制
   */
  const handleImageLoad = useCallback(() => {
    console.log('🖼️ Image loaded, triggering draw')
    drawBoxes()
  }, [drawBoxes])

  /**
   * 窗口大小改变时重绘
   */
  useEffect(() => {
    if (!boxes?.length) return

    const handleResize = () => {
      console.log('📐 Window resized, redrawing')
      drawBoxes()
    }

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [boxes, drawBoxes])

  if (!boxes?.length) {
    return null
  }

  return (
    <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black">
      <img
        ref={imgRef}
        src={imagePreview}
        alt="Result"
        className="w-full block"
        onLoad={handleImageLoad}
      />
      <canvas
        ref={canvasRef}
        className="absolute top-0 left-0 w-full h-full pointer-events-none"
        style={{ display: 'block' }}
      />
    </div>
  )
}

