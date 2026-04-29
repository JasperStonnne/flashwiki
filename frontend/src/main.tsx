import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AuthProvider } from './auth/AuthContext'
import App from './App.tsx'
import './styles/tokens.css'
import './styles/global.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </StrictMode>,
)
