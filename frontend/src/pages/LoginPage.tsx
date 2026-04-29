import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { ApiError, request } from '../api/client'
import { API_ENDPOINTS } from '../api/endpoints'
import { useAuth } from '../auth/AuthContext'
import './login.css'

type Tab = 'login' | 'register'

interface LoginPageProps {
  defaultTab?: Tab
}

type AuthUser = {
  id: string
  email: string
  display_name: string
  role: string
  locale: string
}

type LoginResponse = {
  access_token: string
  refresh_token: string
  user: AuthUser
}

const ERROR_MESSAGES: Record<string, string> = {
  invalid_credentials: '邮箱或密码错误',
  email_taken: '该邮箱已注册',
}

function getErrorMessage(code: string, fallback: string): string {
  return ERROR_MESSAGES[code] ?? fallback
}

export function LoginPage({ defaultTab = 'login' }: LoginPageProps) {
  const [activeTab, setActiveTab] = useState<Tab>(defaultTab)
  const navigate = useNavigate()
  const { login } = useAuth()

  const [loginEmail, setLoginEmail] = useState('')
  const [loginPassword, setLoginPassword] = useState('')

  const [regName, setRegName] = useState('')
  const [regEmail, setRegEmail] = useState('')
  const [regPassword, setRegPassword] = useState('')

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setActiveTab(defaultTab)
  }, [defaultTab])

  function handleTabSwitch(tab: Tab) {
    setActiveTab(tab)
    setError(null)
    navigate(tab === 'login' ? '/login' : '/register', { replace: true })
  }

  async function handleLoginSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const email = loginEmail.trim()
    const password = loginPassword
    if (!email || !password.trim()) {
      setError('请填写邮箱和密码')
      return
    }

    setLoading(true)
    setError(null)
    try {
      const data = await request<LoginResponse>('POST', API_ENDPOINTS.authLogin, {
        email,
        password,
      })
      login(data.access_token, data.refresh_token, data.user.role)
      navigate('/')
    } catch (err) {
      if (err instanceof ApiError) {
        setError(getErrorMessage(err.code, err.message))
      } else {
        setError('登录失败，请稍后重试')
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleRegisterSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const displayName = regName.trim()
    const email = regEmail.trim()
    const password = regPassword

    if (!displayName || !email) {
      setError('请填写昵称和邮箱')
      return
    }
    if (password.length < 8) {
      setError('密码长度至少 8 位')
      return
    }

    setLoading(true)
    setError(null)
    try {
      const data = await request<LoginResponse>('POST', API_ENDPOINTS.authRegister, {
        email,
        password,
        display_name: displayName,
      })
      login(data.access_token, data.refresh_token, data.user.role)
      navigate('/')
    } catch (err) {
      if (err instanceof ApiError) {
        setError(getErrorMessage(err.code, err.message))
      } else {
        setError('注册失败，请稍后重试')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-left">
        <h1>
          让团队知识
          <br />
          <span>流动起来</span>
        </h1>
        <p>
          FPGWiki
          是面向中小团队的在线文档协作平台。实时编辑、精细权限、全文搜索，让每一份知识都触手可及。
        </p>
        <div className="login-features">
          <div className="login-feature">
            <div className="dot" style={{ background: 'var(--primary)' }} />
            实时协同
          </div>
          <div className="login-feature">
            <div className="dot" style={{ background: 'var(--green)' }} />
            权限管理
          </div>
          <div className="login-feature">
            <div className="dot" style={{ background: 'var(--orange)' }} />
            全文搜索
          </div>
        </div>
      </div>

      <div className="login-right">
        <div className="login-card">
          <div className="login-card-head">
            <div className="logo">
              <div className="icon-box">F</div>
              FPGWiki
            </div>
            <p>登录以继续使用</p>
          </div>

          <div className="tabs-pill">
            <button
              type="button"
              className={activeTab === 'login' ? 'active' : ''}
              onClick={() => handleTabSwitch('login')}
            >
              登录
            </button>
            <button
              type="button"
              className={activeTab === 'register' ? 'active' : ''}
              onClick={() => handleTabSwitch('register')}
            >
              注册
            </button>
          </div>

          {error && <div className="form-error">{error}</div>}

          {activeTab === 'login' ? (
            <form id="tab-login" onSubmit={handleLoginSubmit}>
              <div className="form-group">
                <label htmlFor="login-email">邮箱</label>
                <input
                  id="login-email"
                  type="email"
                  placeholder="you@company.com"
                  value={loginEmail}
                  onChange={(event) => setLoginEmail(event.target.value)}
                  disabled={loading}
                />
              </div>
              <div className="form-group">
                <label htmlFor="login-password">密码</label>
                <input
                  id="login-password"
                  type="password"
                  placeholder="输入密码"
                  value={loginPassword}
                  onChange={(event) => setLoginPassword(event.target.value)}
                  disabled={loading}
                />
              </div>
              <button className="login-btn" type="submit" disabled={loading}>
                {loading ? '登录中...' : '登 录'}
              </button>
              <div className="login-footer">
                <a href="#">忘记密码？</a>
              </div>
            </form>
          ) : (
            <form id="tab-register" onSubmit={handleRegisterSubmit}>
              <div className="form-group">
                <label htmlFor="register-name">昵称</label>
                <input
                  id="register-name"
                  type="text"
                  placeholder="你的名字"
                  value={regName}
                  onChange={(event) => setRegName(event.target.value)}
                  disabled={loading}
                />
              </div>
              <div className="form-group">
                <label htmlFor="register-email">邮箱</label>
                <input
                  id="register-email"
                  type="email"
                  placeholder="you@company.com"
                  value={regEmail}
                  onChange={(event) => setRegEmail(event.target.value)}
                  disabled={loading}
                />
              </div>
              <div className="form-group">
                <label htmlFor="register-password">密码</label>
                <input
                  id="register-password"
                  type="password"
                  placeholder="至少 8 位"
                  value={regPassword}
                  onChange={(event) => setRegPassword(event.target.value)}
                  disabled={loading}
                />
              </div>
              <button className="login-btn" type="submit" disabled={loading}>
                {loading ? '注册中...' : '注 册'}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
