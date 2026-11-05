import { createApp } from 'vue'
import App from './App.vue'
import 'vue-sonner/style.css'
import { uploadManager } from './upload/manager'
import { accountOptions, selectedAccount } from './stores/credentials'
import { remoteAlbums } from './stores/albums'
import { watchedFolders } from './stores/watchers'

createApp(App).mount('#app')

if (import.meta.env.DEV) {
  Object.assign(window, {
    uploadManager,
    accountOptions,
    selectedAccount,
    remoteAlbums,
    watchedFolders,
  })
}
