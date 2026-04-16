<script setup>
import { ref, onMounted } from 'vue'
import { Pack, Extract, Repair, Test, OpenFileDialog, OpenDirectoryDialog, GetFileInfo, Version, GetStartupFile, GetStartupAction } from '../wailsjs/go/main/App'

const currentView = ref('home')
const selectedPath = ref('')
const fileInfo = ref(null)
const dragOver = ref(false)
const loading = ref(false)
const result = ref(null)

// Pack options
const format = ref('nya')
const level = ref(9)
const fec = ref(10)
const password = ref('')
const solid = ref(false)
const sfx = ref(false)

// Handle startup file (double-click or right-click menu)
onMounted(async () => {
  const file = await GetStartupFile()
  const action = await GetStartupAction()
  if (file) {
    await selectPath(file)
    if (action === 'extract') { doExtract() }
    else if (action === 'repair') { currentView.value = 'repair'; doRepair() }
    else if (action === 'pack') { doPack() }
  }
})

async function browseFile() {
  const path = await OpenFileDialog()
  if (path) { await selectPath(path) }
}

async function browseFolder() {
  const path = await OpenDirectoryDialog()
  if (path) { await selectPath(path) }
}

async function selectPath(path) {
  selectedPath.value = path
  const info = await GetFileInfo(path)
  fileInfo.value = info
  result.value = null
}

function formatSize(bytes) {
  if (!bytes) return '0B'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024*1024) return (bytes/1024).toFixed(1) + 'KB'
  if (bytes < 1024*1024*1024) return (bytes/1024/1024).toFixed(1) + 'MB'
  return (bytes/1024/1024/1024).toFixed(2) + 'GB'
}

async function doPack() {
  loading.value = true
  result.value = null
  try {
    const r = await Pack({
      input: selectedPath.value,
      format: format.value,
      level: parseInt(level.value),
      fec: parseInt(fec.value),
      password: password.value,
      solid: solid.value,
      sfx: sfx.value,
    })
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

async function doExtract() {
  loading.value = true
  result.value = null
  try {
    const r = await Extract(selectedPath.value)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

async function doRepair() {
  loading.value = true
  result.value = null
  try {
    const r = await Repair(selectedPath.value)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

async function doTest() {
  loading.value = true
  result.value = null
  try {
    const r = await Test(selectedPath.value)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

function clear() {
  selectedPath.value = ''
  fileInfo.value = null
  result.value = null
}
</script>

<template>
  <div class="flex flex-col h-screen bg-gray-950 text-white">
    <!-- Header -->
    <header class="bg-gray-900 border-b border-gray-800 px-6 py-3 flex items-center justify-between select-none" style="--wails-draggable:drag">
      <div class="flex items-center gap-3">
        <span class="text-2xl">🐱</span>
        <h1 class="text-xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">NekoArc</h1>
        <span class="text-xs text-gray-600">v0.1.0</span>
      </div>
      <nav class="flex gap-1" style="--wails-draggable:no-drag">
        <button v-for="v in [{id:'home',icon:'📦',label:'Pack'},{id:'repair',icon:'🔧',label:'Repair'},{id:'about',icon:'ℹ️',label:'About'}]"
          :key="v.id" @click="currentView=v.id"
          :class="currentView===v.id ? 'bg-purple-600' : 'bg-gray-800 hover:bg-gray-700'"
          class="px-4 py-1.5 rounded-lg text-sm font-medium transition">
          {{ v.icon }} {{ v.label }}
        </button>
      </nav>
    </header>

    <main class="flex-1 p-6 overflow-auto">
      <!-- HOME -->
      <div v-if="currentView==='home'" class="max-w-3xl mx-auto space-y-5">
        
        <!-- File Selection -->
        <div v-if="!fileInfo" class="border-2 border-dashed rounded-2xl p-12 text-center transition-all"
             :class="dragOver ? 'border-purple-500 bg-purple-500/5' : 'border-gray-700 hover:border-gray-600'">
          <div class="text-5xl mb-4">📁</div>
          <p class="text-lg font-medium text-gray-300 mb-4">Select files to pack or extract</p>
          <div class="flex gap-3 justify-center">
            <button @click="browseFile" class="bg-purple-600 hover:bg-purple-500 px-6 py-2.5 rounded-xl text-sm font-medium transition">
              📄 Open File
            </button>
            <button @click="browseFolder" class="bg-gray-700 hover:bg-gray-600 px-6 py-2.5 rounded-xl text-sm font-medium transition">
              📁 Open Folder
            </button>
          </div>
          <p class="text-xs text-gray-600 mt-4">Supports: .nya .zip .rar .7z .tar .gz .bz2 .xz</p>
        </div>

        <!-- Selected File -->
        <div v-if="fileInfo" class="bg-gray-900 rounded-xl p-5">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <span class="text-2xl">{{ fileInfo.isDir ? '📁' : '📄' }}</span>
              <div>
                <p class="font-medium">{{ fileInfo.name }}</p>
                <p class="text-xs text-gray-500">{{ fileInfo.path }} · {{ formatSize(fileInfo.size) }}</p>
              </div>
            </div>
            <button @click="clear" class="text-gray-500 hover:text-red-400 px-3 py-1 text-sm">✕ Clear</button>
          </div>
        </div>

        <!-- Pack Options -->
        <div v-if="fileInfo" class="bg-gray-900 rounded-xl p-6 space-y-4">
          <h3 class="font-semibold text-gray-400 text-xs uppercase tracking-wider">Pack Options</h3>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="text-xs text-gray-500 block mb-1.5">Format</label>
              <select v-model="format" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:border-purple-500 outline-none">
                <option value="nya">.nya (FEC Protected)</option>
                <option value="zip">.zip</option>
                <option value="tar.gz">.tar.gz</option>
                <option value="rar">.rar (Store) ⚠️ Experimental</option>
              </select>
            </div>
            <div>
              <label class="text-xs text-gray-500 block mb-1.5">Level</label>
              <div class="flex items-center gap-3">
                <input type="range" v-model="level" min="1" max="19" class="flex-1 accent-purple-500">
                <span class="text-sm text-gray-400 w-6 text-right">{{ level }}</span>
              </div>
            </div>
            <div v-if="format==='nya'">
              <label class="text-xs text-gray-500 block mb-1.5">FEC Recovery %</label>
              <div class="flex items-center gap-3">
                <input type="range" v-model="fec" min="0" max="100" class="flex-1 accent-green-500">
                <span class="text-sm text-gray-400 w-10 text-right">{{ fec }}%</span>
              </div>
            </div>
            <div>
              <label class="text-xs text-gray-500 block mb-1.5">Password</label>
              <input type="password" v-model="password" placeholder="Optional"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:border-purple-500 outline-none">
            </div>
          </div>
          <div class="flex gap-6">
            <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
              <input type="checkbox" v-model="solid" class="accent-purple-500"> Solid
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
              <input type="checkbox" v-model="sfx" class="accent-purple-500"> SFX
            </label>
          </div>

          <div class="flex gap-3 pt-1">
            <button @click="doPack" :disabled="loading"
              class="flex-1 bg-purple-600 hover:bg-purple-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ loading ? '⏳ Working...' : '📦 Pack' }}
            </button>
            <button @click="doExtract" :disabled="loading"
              class="flex-1 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ loading ? '⏳ Working...' : '📂 Extract' }}
            </button>
          </div>
        </div>

        <!-- Result -->
        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <div class="flex justify-between items-start">
            <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">
              {{ result.success ? '✅ Success' : '❌ Error' }}
            </p>
            <span v-if="result.duration" class="text-xs text-gray-500">{{ result.duration.toFixed(2) }}s</span>
          </div>
          <pre class="text-xs text-gray-400 mt-2 whitespace-pre-wrap">{{ result.message }}</pre>
        </div>
      </div>

      <!-- REPAIR -->
      <div v-if="currentView==='repair'" class="max-w-3xl mx-auto space-y-5">
        <div v-if="!fileInfo" class="border-2 border-dashed border-gray-700 rounded-2xl p-16 text-center hover:border-green-600 transition-colors">
          <div class="text-6xl mb-4">🔧</div>
          <p class="text-xl font-medium text-gray-300 mb-2">Repair a damaged .nya archive</p>
          <p class="text-sm text-gray-500 mb-6">RaptorQ FEC recovers up to <span class="text-green-400 font-semibold">50% damage</span></p>
          <button @click="browseFile" class="bg-green-600 hover:bg-green-500 px-8 py-3 rounded-xl font-medium transition">
            📄 Select Archive
          </button>
        </div>

        <div v-if="fileInfo" class="bg-gray-900 rounded-xl p-5">
          <div class="flex items-center gap-3">
            <span class="text-2xl">📄</span>
            <div>
              <p class="font-medium">{{ fileInfo.name }}</p>
              <p class="text-xs text-gray-500">{{ formatSize(fileInfo.size) }}</p>
            </div>
          </div>
        </div>

        <div v-if="fileInfo" class="flex gap-3">
          <button @click="doTest" :disabled="loading"
            class="flex-1 bg-yellow-600 hover:bg-yellow-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
            {{ loading ? '⏳...' : '🔍 Test' }}
          </button>
          <button @click="doRepair" :disabled="loading"
            class="flex-1 bg-green-600 hover:bg-green-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
            {{ loading ? '⏳...' : '🔧 Repair' }}
          </button>
        </div>

        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">
            {{ result.success ? '✅ Done' : '❌ Failed' }}
          </p>
          <pre class="text-xs text-gray-400 mt-2 whitespace-pre-wrap">{{ result.message }}</pre>
        </div>
      </div>

      <!-- ABOUT -->
      <div v-if="currentView==='about'" class="max-w-3xl mx-auto space-y-5">
        <div class="bg-gray-900 rounded-xl p-8 text-center">
          <div class="text-6xl mb-4">🐱</div>
          <h2 class="text-3xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">NekoArc</h2>
          <p class="text-gray-400 mt-2">Next-generation archive manager with self-healing FEC</p>
        </div>
        <div class="bg-gray-900 rounded-xl p-6 grid grid-cols-2 gap-y-4 text-sm">
          <div><span class="text-gray-500">Core</span><br><span class="text-gray-300">Nyarc v0.6.0</span></div>
          <div><span class="text-gray-500">FEC Engine</span><br><span class="text-gray-300">GoFEC (RaptorQ + LDPC)</span></div>
          <div><span class="text-gray-500">Compression</span><br><span class="text-gray-300">Zstd 1-19</span></div>
          <div><span class="text-gray-500">Hash</span><br><span class="text-gray-300">BLAKE3</span></div>
          <div><span class="text-gray-500">Encryption</span><br><span class="text-gray-300">AES-256-GCM</span></div>
          <div><span class="text-gray-500">Max Recovery</span><br><span class="text-green-400 font-bold">50%</span></div>
        </div>
        <div class="bg-gray-900 rounded-xl p-6 text-sm space-y-2">
          <h3 class="font-semibold text-gray-400 text-xs uppercase tracking-wider mb-3">Supported Formats</h3>
          <div class="grid grid-cols-2 gap-2 text-gray-400">
            <div>📦 <span class="text-purple-400">.nya</span> — Pack + Extract + Repair</div>
            <div>📦 .zip — Pack + Extract</div>
            <div>📦 .tar.gz — Pack + Extract</div>
            <div>📦 .rar — Pack (Store) + Extract</div>
            <div>📦 .7z — Extract</div>
            <div>📦 .tar.bz2 / .tar.xz — Pack + Extract</div>
          </div>
        </div>
        <p class="text-center text-xs text-gray-600">© 2026 Nyarime · github.com/Nyarime/Nyarc</p>
      </div>
    </main>

    <footer class="bg-gray-900 border-t border-gray-800 px-6 py-2 flex justify-between text-xs text-gray-500">
      <span>NekoArc v0.1.0 — Nyarc v0.6.0</span>
      <span>RaptorQ • BLAKE3 • AES-256-GCM • 50% Recovery</span>
    </footer>
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
html, body, #app { margin:0; padding:0; height:100%; font-family:'Inter',system-ui,sans-serif; background:#030712; }
</style>
