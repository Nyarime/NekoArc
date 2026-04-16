<script setup>
import { ref, onMounted } from 'vue'
import { Pack, EstimateSize, Extract, Repair, Test, OpenFileDialog, OpenMultipleFilesDialog, OpenDirectoryDialog, GetFileInfo, Version, GetStartupFile, GetStartupAction } from '../wailsjs/go/main/App'

const currentView = ref('home')
const loading = ref(false)
const result = ref(null)

// Pack state
const packFiles = ref([])  // [{name, path, size}]
const packDir = ref('')
const format = ref('nya')
const level = ref(9)
const fec = ref(100)
const password = ref('')
const solid = ref(false)
const sfx = ref(false)
const estimate = ref(null)

// Extract state
const extractFile = ref(null)
const extractDir = ref('')

function formatSize(bytes) {
  if (!bytes) return '0B'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024*1024) return (bytes/1024).toFixed(1) + 'KB'
  if (bytes < 1024*1024*1024) return (bytes/1024/1024).toFixed(1) + 'MB'
  return (bytes/1024/1024/1024).toFixed(2) + 'GB'
}

// Pack: add files
async function addFiles() {
  const paths = await OpenMultipleFilesDialog()
  if (paths && paths.length > 0) {
    for (const p of paths) {
      const info = await GetFileInfo(p)
      if (info && !packFiles.value.find(f => f.path === p)) {
        packFiles.value.push(info)
      }
    }
  }
  await updateEstimate()
}

async function addFolder() {
  const p = await OpenDirectoryDialog()
  if (p) {
    const info = await GetFileInfo(p)
    if (info && !packFiles.value.find(f => f.path === p)) {
      packFiles.value.push(info)
    }
  }
}

async function updateEstimate() {
  if (packFiles.value.length > 0) {
    const paths = packFiles.value.map(f => f.path)
    estimate.value = await EstimateSize(paths, parseInt(fec.value))
  } else {
    estimate.value = null
  }
}

function removePackFile(f) {
  packFiles.value = packFiles.value.filter(x => x.path !== f.path)
}

function clearPack() {
  packFiles.value = []
  packDir.value = ''
  result.value = null
}

async function choosePackDir() {
  const p = await OpenDirectoryDialog()
  if (p) packDir.value = p
}

async function doPack() {
  if (packFiles.value.length === 0) return
  loading.value = true
  result.value = null
  try {
    // Pack first item (TODO: multi-file pack)
    const r = await Pack({
      inputs: packFiles.value.map(f => f.path),
      output: packDir.value,
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

// Extract
async function chooseExtractFile() {
  const p = await OpenNyaFileDialog()
  if (p) {
    extractFile.value = await GetFileInfo(p)
  }
}

async function chooseExtractDir() {
  const p = await OpenDirectoryDialog()
  if (p) extractDir.value = p
}

function clearExtract() {
  extractFile.value = null
  extractDir.value = ''
  result.value = null
}

async function doExtract() {
  if (!extractFile.value) return
  loading.value = true
  result.value = null
  try {
    const r = await Extract(extractFile.value.path, extractDir.value)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

// Repair
const repairFile = ref(null)

async function chooseRepairFile() {
  const { OpenNyaFileDialog } = await import("../wailsjs/go/main/App")
  const p = await OpenNyaFileDialog()
  if (p) repairFile.value = await GetFileInfo(p)
}

async function doRepair() {
  if (!repairFile.value) return
  loading.value = true
  result.value = null
  try {
    const r = await Repair(repairFile.value.path)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

async function doTest() {
  if (!repairFile.value) return
  loading.value = true
  result.value = null
  try {
    const r = await Test(repairFile.value.path)
    result.value = r
  } catch(e) {
    result.value = { success: false, message: String(e), duration: 0 }
  }
  loading.value = false
}

onMounted(async () => {
  const file = await GetStartupFile()
  const action = await GetStartupAction()
  if (file) {
    const info = await GetFileInfo(file)
    if (action === 'extract') { extractFile.value = info; currentView.value = 'extract'; doExtract() }
    else if (action === 'repair') { repairFile.value = info; currentView.value = 'repair'; doRepair() }
    else { packFiles.value = [info] }
  }
})
</script>

<template>
  <div class="flex flex-col h-screen bg-gray-950 text-white">
    <header class="bg-gray-900 border-b border-gray-800 px-6 py-3 flex items-center justify-between select-none" style="--wails-draggable:drag">
      <div class="flex items-center gap-3">
        <span class="text-2xl">🐱</span>
        <h1 class="text-xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">NekoArc</h1>
        <span class="text-xs text-gray-600">v0.1.0</span>
      </div>
      <nav class="flex gap-1" style="--wails-draggable:no-drag">
        <button v-for="v in [{id:'home',icon:'📦',label:'Pack'},{id:'extract',icon:'📂',label:'Extract'},{id:'repair',icon:'🔧',label:'Repair'},{id:'about',icon:'ℹ️',label:'About'}]"
          :key="v.id" @click="currentView=v.id; result=null"
          :class="currentView===v.id ? 'bg-purple-600' : 'bg-gray-800 hover:bg-gray-700'"
          class="px-4 py-1.5 rounded-lg text-sm font-medium transition">
          {{ v.icon }} {{ v.label }}
        </button>
      </nav>
    </header>

    <main class="flex-1 p-6 overflow-auto">

      <!-- PACK -->
      <div v-if="currentView==='home'" class="max-w-3xl mx-auto space-y-4">
        <!-- Add files -->
        <div class="flex gap-3">
          <button @click="addFiles" class="flex-1 bg-purple-600 hover:bg-purple-500 py-3 rounded-xl text-sm font-medium transition">
            📄 Add Files
          </button>
          <button @click="addFolder" class="flex-1 bg-gray-700 hover:bg-gray-600 py-3 rounded-xl text-sm font-medium transition">
            📁 Add Folder
          </button>
          <button v-if="packFiles.length" @click="clearPack" class="bg-red-900 hover:bg-red-800 px-4 py-3 rounded-xl text-sm transition">
            ✕
          </button>
        </div>

        <!-- File list -->
        <div v-if="packFiles.length" class="bg-gray-900 rounded-xl p-3 space-y-1 max-h-48 overflow-auto">
          <div v-for="f in packFiles" :key="f.path"
               class="flex items-center justify-between bg-gray-800 rounded-lg px-3 py-2">
            <div class="flex items-center gap-2 min-w-0">
              <span>{{ f.isDir ? '📁' : '📄' }}</span>
              <span class="text-sm truncate">{{ f.name }}</span>
              <span class="text-xs text-gray-500 shrink-0">{{ formatSize(f.size) }}</span>
            </div>
            <button @click="removePackFile(f)" class="text-red-400 hover:text-red-300 text-xs px-2 shrink-0">✕</button>
          </div>
        </div>

        <!-- Options -->
        <div v-if="packFiles.length" class="bg-gray-900 rounded-xl p-5 space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="text-xs text-gray-500 block mb-1">Format</label>
              <select v-model="format" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:border-purple-500 outline-none">
                <option value="nya">.nya (FEC Protected)</option>
                <option value="zip">.zip</option>
                <option value="tar.gz">.tar.gz</option>
                <option value="rar">.rar ⚠️</option>
              </select>
            </div>
            <div>
              <label class="text-xs text-gray-500 block mb-1">Level</label>
              <div class="flex items-center gap-2">
                <input type="range" v-model="level" min="1" max="19" class="flex-1 accent-purple-500">
                <span class="text-sm text-gray-400 w-6 text-right">{{ level }}</span>
              </div>
            </div>
            <div v-if="format==='nya'">
              <label class="text-xs text-gray-500 block mb-1">FEC %</label>
              <div class="flex items-center gap-2">
                <input type="range" v-model="fec" min="0" max="100" class="flex-1 accent-green-500">
                <span class="text-sm text-gray-400 w-8 text-right">{{ fec }}%</span>
              </div>
            </div>
            <div>
              <label class="text-xs text-gray-500 block mb-1">Password</label>
              <input type="password" v-model="password" placeholder="Optional"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm outline-none">
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

          <!-- Save to -->
          <div class="flex items-center justify-between">
            <div>
              <label class="text-xs text-gray-500 block mb-0.5">Save to</label>
              <p class="text-sm text-gray-300">{{ packDir || 'Same directory' }}</p>
            </div>
            <button @click="choosePackDir" class="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg text-xs transition">📁 Browse</button>
          </div>

          <!-- Estimate -->
          <div v-if="estimate" class="bg-gray-800 rounded-lg p-3 text-xs text-gray-400 flex justify-between">
            <span>Input: {{ formatSize(estimate.inputSize) }}</span>
            <span>Est. output: {{ formatSize(estimate.outputSize) }}</span>
            <span>FEC: {{ formatSize(estimate.fecSize) }}</span>
            <span class="text-green-400">Recovery: {{ estimate.recoveryRate }}</span>
          </div>

          <button @click="doPack" :disabled="loading"
            class="w-full bg-purple-600 hover:bg-purple-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
            {{ loading ? '⏳ Packing...' : '📦 Pack' }}
          </button>
        </div>

        <div v-if="!packFiles.length" class="border-2 border-dashed border-gray-700 rounded-2xl p-12 text-center">
          <div class="text-5xl mb-4">📦</div>
          <p class="text-gray-400">Click <b>Add Files</b> or <b>Add Folder</b> to start</p>
        </div>

        <!-- Result -->
        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <div class="flex justify-between">
            <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">
              {{ result.success ? '✅ Success' : '❌ Error' }}
            </p>
            <span v-if="result.duration" class="text-xs text-gray-500">{{ result.duration.toFixed(2) }}s</span>
          </div>
          <pre class="text-xs text-gray-400 mt-1 whitespace-pre-wrap">{{ result.message }}</pre>
        </div>
      </div>

      <!-- EXTRACT -->
      <div v-if="currentView==='extract'" class="max-w-3xl mx-auto space-y-4">
        <div v-if="!extractFile" class="border-2 border-dashed border-gray-700 rounded-2xl p-12 text-center hover:border-blue-600 transition-colors">
          <div class="text-5xl mb-4">📂</div>
          <p class="text-lg text-gray-300 mb-4">Select an archive to extract</p>
          <button @click="chooseExtractFile" class="bg-blue-600 hover:bg-blue-500 px-6 py-2.5 rounded-xl text-sm font-medium transition">
            📄 Select Archive
          </button>
          <p class="text-xs text-gray-600 mt-3">.nya .zip .rar .7z .tar .gz .bz2 .xz</p>
        </div>

        <div v-if="extractFile" class="space-y-3">
          <div class="bg-gray-900 rounded-xl p-4 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <span>📄</span>
              <div>
                <p class="font-medium text-sm">{{ extractFile.name }}</p>
                <p class="text-xs text-gray-500">{{ formatSize(extractFile.size) }}</p>
              </div>
            </div>
            <button @click="clearExtract" class="text-gray-500 hover:text-red-400 text-sm">✕</button>
          </div>

          <div class="bg-gray-900 rounded-xl p-4 flex items-center justify-between">
            <div>
              <label class="text-xs text-gray-500 block mb-0.5">Extract to</label>
              <p class="text-sm text-gray-300">{{ extractDir || 'Same directory' }}</p>
            </div>
            <button @click="chooseExtractDir" class="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg text-xs transition">📁 Browse</button>
          </div>

          <button @click="doExtract" :disabled="loading"
            class="w-full bg-blue-600 hover:bg-blue-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
            {{ loading ? '⏳ Extracting...' : '📂 Extract' }}
          </button>
        </div>

        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">{{ result.message }}</p>
        </div>
      </div>

      <!-- REPAIR -->
      <div v-if="currentView==='repair'" class="max-w-3xl mx-auto space-y-4">
        <div v-if="!repairFile" class="border-2 border-dashed border-gray-700 rounded-2xl p-12 text-center hover:border-green-600 transition-colors">
          <div class="text-5xl mb-4">🔧</div>
          <p class="text-lg text-gray-300 mb-2">Repair a damaged .nya archive</p>
          <p class="text-sm text-gray-500 mb-4">RaptorQ FEC recovers up to <span class="text-green-400 font-semibold">66%+ damage</span></p>
          <button @click="chooseRepairFile" class="bg-green-600 hover:bg-green-500 px-6 py-2.5 rounded-xl text-sm font-medium transition">
            📄 Select .nya File
          </button>
        </div>

        <div v-if="repairFile" class="space-y-3">
          <div class="bg-gray-900 rounded-xl p-4 flex items-center gap-3">
            <span>📄</span>
            <div><p class="font-medium text-sm">{{ repairFile.name }}</p><p class="text-xs text-gray-500">{{ formatSize(repairFile.size) }}</p></div>
          </div>
          <div class="flex gap-3">
            <button @click="doTest" :disabled="loading"
              class="flex-1 bg-yellow-600 hover:bg-yellow-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ loading ? '⏳...' : '🔍 Test' }}
            </button>
            <button @click="doRepair" :disabled="loading"
              class="flex-1 bg-green-600 hover:bg-green-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ loading ? '⏳...' : '🔧 Repair' }}
            </button>
          </div>
        </div>

        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">{{ result.message }}</p>
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
          <div><span class="text-gray-500">Core</span><br><span class="text-gray-300">Nyarc v0.6.2</span></div>
          <div><span class="text-gray-500">FEC</span><br><span class="text-gray-300">GoFEC v1.2.0 (RaptorQ)</span></div>
          <div><span class="text-gray-500">Compression</span><br><span class="text-gray-300">Zstd 1-19</span></div>
          <div><span class="text-gray-500">Hash</span><br><span class="text-gray-300">BLAKE3</span></div>
          <div><span class="text-gray-500">Encryption</span><br><span class="text-gray-300">AES-256-GCM</span></div>
          <div><span class="text-gray-500">Max Recovery</span><br><span class="text-green-400 font-bold">Up to 66%+</span></div>
        </div>
        <p class="text-center text-xs text-gray-600">© 2026 Nyarime · github.com/Nyarime/Nyarc</p>
      </div>
    </main>

    <footer class="bg-gray-900 border-t border-gray-800 px-6 py-2 flex justify-between text-xs text-gray-500">
      <span>NekoArc v0.1.0 — Nyarc v0.6.2</span>
      <span>RaptorQ • BLAKE3 • AES-256-GCM</span>
    </footer>
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
html, body, #app { margin:0; padding:0; height:100%; font-family:'Inter',system-ui,sans-serif; background:#030712; }
</style>
