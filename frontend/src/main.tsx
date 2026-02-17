import React from 'react'
import { createRoot } from 'react-dom/client'
import './index.css' // Updated to use TUI styles
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App />
    </React.StrictMode>
)
