/**
 * OCR 状态管理 Hook
 * 封装 OCR 相关的状态和逻辑
 */
import { useState, useCallback } from 'react'
import { ocrClient } from '../api/client'

/**
 * @typedef {Object} OCRState
 * @property {File|null} image - 上传的图像文件
 * @property {string|null} imagePreview - 图像预览 URL
 * @property {Object|null} result - OCR 结果
 * @property {boolean} loading - 是否正在加载
 * @property {string|null} error - 错误信息
 */

/**
 * @typedef {Object} OCRActions
 * @property {Function} handleImageSelect - 选择图像
 * @property {Function} handleSubmit - 提交 OCR 请求
 * @property {Function} clearError - 清除错误
 * @property {Function} reset - 重置所有状态
 */

/**
 * useOCR Hook
 * @returns {[OCRState, OCRActions]}
 */
export default function useOCR() {
  const [image, setImage] = useState(null)
  const [imagePreview, setImagePreview] = useState(null)
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  /**
   * 处理图像选择
   */
  const handleImageSelect = useCallback((file) => {
    if (file === null) {
      // 清除图像
      setImage(null)
      if (imagePreview) {
        URL.revokeObjectURL(imagePreview)
      }
      setImagePreview(null)
      setError(null)
      setResult(null)
    } else {
      // 设置新图像
      setImage(file)
      setImagePreview(URL.createObjectURL(file))
      setError(null)
      setResult(null)
    }
  }, [imagePreview])

  /**
   * 提交 OCR 请求
   */
  const handleSubmit = useCallback(async (params) => {
    if (!image) {
      setError('Please upload an image first')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const inferResult = await ocrClient.infer({
        image,
        ...params,
      })
      setResult(inferResult)
    } catch (err) {
      setError(err.message || 'An error occurred')
    } finally {
      setLoading(false)
    }
  }, [image])

  /**
   * 清除错误
   */
  const clearError = useCallback(() => {
    setError(null)
  }, [])

  /**
   * 重置所有状态
   */
  const reset = useCallback(() => {
    setImage(null)
    if (imagePreview) {
      URL.revokeObjectURL(imagePreview)
    }
    setImagePreview(null)
    setResult(null)
    setError(null)
    setLoading(false)
  }, [imagePreview])

  return [
    {
      image,
      imagePreview,
      result,
      loading,
      error,
    },
    {
      handleImageSelect,
      handleSubmit,
      clearError,
      reset,
    },
  ]
}

