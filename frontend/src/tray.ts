import { createApp } from 'vue'
import TrayFlyout from './views/TrayFlyout.vue'
import './index.css'

document.documentElement.classList.add('dark')
createApp(TrayFlyout).mount('#app')
