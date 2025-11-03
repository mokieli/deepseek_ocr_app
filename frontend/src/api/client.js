/**
 * API 客户端
 * 封装所有 API 调用
 */
import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_URL || '/api'

/**
 * OCR API 客户端类
 */
class OCRClient {
  constructor(baseURL = API_BASE) {
    this.client = axios.create({
      baseURL: baseURL.replace('/api', ''),
      timeout: 300000, // 5 分钟超时
    })

    // 响应拦截器
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        const errorMessage = error.response?.data?.detail || error.message || 'An error occurred'
        console.error('API Error:', errorMessage)
        throw new Error(errorMessage)
      }
    )
  }

  /**
   * 健康检查
   */
  async healthCheck() {
    const response = await this.client.get('/health')
    return response.data
  }

  /**
   * OCR 推理
   * @param {Object} params - 推理参数
   * @returns {Promise<Object>} 推理结果
   */
  async infer(params) {
    const {
      image,
      mode = 'plain_ocr',
      prompt = '',
      grounding = false,
      includeCaption = false,
      findTerm = '',
      schema = '',
      baseSize = 1024,
      imageSize = 640,
      cropMode = true,
      testCompress = false,
    } = params

    if (!image) {
      throw new Error('Image is required')
    }

    const formData = new FormData()
    formData.append('image', image)
    formData.append('mode', mode)
    formData.append('prompt', prompt)
    formData.append('grounding', grounding)
    formData.append('include_caption', includeCaption)
    formData.append('find_term', findTerm)
    formData.append('schema', schema)
    formData.append('base_size', baseSize)
    formData.append('image_size', imageSize)
    formData.append('crop_mode', cropMode)
    formData.append('test_compress', testCompress)

    const response = await this.client.post('/api/ocr', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    return response.data
  }
}

// 导出单例实例
export const ocrClient = new OCRClient()

// 导出类以便测试
export default OCRClient

