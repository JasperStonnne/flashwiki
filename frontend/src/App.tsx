import { useEffect } from 'react'
import axios from 'axios'

function App() {
  useEffect(() => {
    // 尝试去 8080 端口找后端打招呼
    axios.get('http://localhost:8080/ping')
      .then(res => alert("连接成功: " + res.data.message))
      .catch(err => alert("连接失败，请确认后端 8080 是否启动！"));
  }, [])

  return (
    <div style={{ padding: '40px' }}>
      <h1>FlashWiki 开发环境已就绪</h1>
      <p>如果看到弹窗，说明前后端已经打通了！</p>
    </div>
  )
}

export default App