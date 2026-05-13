export const headers = {
  'Content-Type': 'application/json',
  'X-Requested-With': 'XMLHttpRequest',
};

/**
 * 通用请求方法
 * @param {string} method - HTTP 方法 (GET, POST, PUT, DELETE 等)
 * @param {string} url - 请求 URL
 * @param {Object|null} data - 请求数据 (GET 请求可为 null)
 * @returns {Promise<Object>} 返回解析后的 JSON 数据
 */
function request(method, url, data = null) {
  // 构建请求配置
  const config = {
    method,
    headers,
  };

  // 如果有数据且不是 GET 请求，则添加 body
  if (data && method.toUpperCase() !== 'GET') {
    config.body = JSON.stringify(data);
  }

  // 发起请求
  return fetch(url, config)
    .then(function (res) {
      // 检查响应状态码
      if (res.status >= 400) {
        throw new Error(`HTTP error! status: ${res.status}`);
      }

      // 尝试解析 JSON 响应
      return res.json().catch((e) => {
        // 如果解析失败，返回空对象
        console.warn('解析json失败', e);
        return { success: false, message: e.message || '请求解析为json失败' };
      });
    })
    .then(function (json) {
      // 可以在这里添加统一的业务逻辑处理
      // 例如：检查后端返回的业务状态码
      if (json.success) {
        return json.data;
      }
      throw new Error(JSON.stringify(json));
    });
}

export default {
  /**
   * GET 请求
   * @param {string} url - 请求 URL
   * @returns {Promise<Object>}
   */
  get(url) {
    return request('GET', url);
  },

  /**
   * POST 请求
   * @param {string} url - 请求 URL
   * @param {Object} data - 请求数据
   * @returns {Promise<Object>}
   */
  post(url, data) {
    return request('POST', url, data);
  },

  /**
   * PUT 请求
   * @param {string} url - 请求 URL
   * @param {Object} data - 请求数据
   * @returns {Promise<Object>}
   */
  put(url, data) {
    return request('PUT', url, data);
  },

  /**
   * DELETE 请求
   * @param {string} url - 请求 URL
   * @returns {Promise<Object>}
   */
  delete(url) {
    return request('DELETE', url);
  },
};
