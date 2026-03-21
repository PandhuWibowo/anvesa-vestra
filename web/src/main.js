import { createApp } from 'vue'
import App from './App.vue'
import './styles.css'
import naive from 'naive-ui'

const app = createApp(App)
app.config.errorHandler = (err, vm, info) => {
  console.error('[Vue Error]', err, info)
}
app.use(naive)
app.mount('#app')
