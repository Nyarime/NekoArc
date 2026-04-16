<script setup>
import { ref } from 'vue'
import { Greet } from '../wailsjs/go/main/App'

const currentView = ref('home')
const files = ref([])
const dragOver = ref(false)
const packing = ref(false)
const extracting = ref(false)
const repairing = ref(false)
const result = ref(null)

// Pack options
const format = ref('nya')
const level = ref(9)
const fec = ref(10)
const password = ref('')
const solid = ref(false)
const sfx = ref(false)

function onDrop(e) {
  dragOver.value = false
  for (let f of e.dataTransfer.files) {
    files.value.push({ name: f.name, size: f.size })
  }
}

function removeFile(f) {
  files.value = files.value.filter(x => x !== f)
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024*1024) return (bytes/1024).toFixed(1) + 'KB'
  if (bytes < 1024*1024*1024) return (bytes/1024/1024).toFixed(1) + 'MB'
  return (bytes/1024/1024/1024).toFixed(2) + 'GB'
}

async function pack() {
  packing.value = true
  result.value = null
  try {
    const msg = await Greet('Pack')
    result.value = { success: true, message: msg }
  } catch(e) {
    result.value = { success: false, message: e }
  }
  packing.value = false
}

async function extract() {
  extracting.value = true
  result.value = null
  try {
    const msg = await Greet('Extract')
    result.value = { success: true, message: msg }
  } catch(e) {
    result.value = { success: false, message: e }
  }
  extracting.value = false
}

async function repair() {
  repairing.value = true
  result.value = null
  try {
    const msg = await Greet('Repair')
    result.value = { success: true, message: msg }
  } catch(e) {
    result.value = { success: false, message: e }
  }
  repairing.value = false
}
</script>

<template>
  <div class="flex flex-col h-screen bg-gray-950 text-white">
    <!-- Header -->
    <header class="bg-gray-900 border-b border-gray-800 px-6 py-3 flex items-center justify-between select-none" style="--wails-draggable:drag">
      <div class="flex items-center gap-3">
        <span class="text-2xl">🐱</span>
        <h1 class="text-xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">
          NekoArc
        </h1>
        <span class="text-xs text-gray-500">v0.1.0</span>
      </div>
      <nav class="flex gap-1" style="--wails-draggable:no-drag">
        <button @click="currentView='home'"
          :class="currentView==='home' ? 'bg-purple-600' : 'bg-gray-800 hover:bg-gray-700'"
          class="px-4 py-1.5 rounded-lg text-sm font-medium transition">
          📦 Pack
        </button>
        <button @click="currentView='repair'"
          :class="currentView==='repair' ? 'bg-purple-600' : 'bg-gray-800 hover:bg-gray-700'"
          class="px-4 py-1.5 rounded-lg text-sm font-medium transition">
          🔧 Repair
        </button>
        <button @click="currentView='about'"
          :class="currentView==='about' ? 'bg-purple-600' : 'bg-gray-800 hover:bg-gray-700'"
          class="px-4 py-1.5 rounded-lg text-sm font-medium transition">
          ℹ️ About
        </button>
      </nav>
    </header>

    <!-- Main -->
    <main class="flex-1 p-6 overflow-auto">

      <!-- Home: Pack & Extract -->
      <div v-if="currentView==='home'" class="max-w-3xl mx-auto space-y-6">
        <!-- Drop Zone -->
        <div class="border-2 border-dashed rounded-2xl p-12 text-center cursor-pointer transition-all duration-200"
             :class="dragOver ? 'border-purple-500 bg-purple-500/5' : 'border-gray-700 hover:border-gray-600'"
             @dragover.prevent="dragOver=true"
             @dragleave="dragOver=false"
             @drop.prevent="onDrop">
          <div class="text-5xl mb-4">{{ dragOver ? '✨' : '📁' }}</div>
          <p class="text-lg font-medium text-gray-300">Drag files or folders here</p>
          <p class="text-sm text-gray-500 mt-2">or click to browse</p>
          <p class="text-xs text-gray-600 mt-4">
            Supports: .nya .zip .rar .7z .tar .gz .bz2 .xz
          </p>
        </div>

        <!-- File List -->
        <div v-if="files.length" class="bg-gray-900 rounded-xl p-4 space-y-2">
          <div v-for="(f, i) in files" :key="i"
               class="flex items-center justify-between bg-gray-800 rounded-lg px-4 py-2.5">
            <div class="flex items-center gap-3">
              <span>📄</span>
              <span class="font-medium text-sm">{{ f.name }}</span>
              <span class="text-xs text-gray-500">{{ formatSize(f.size) }}</span>
            </div>
            <button @click="removeFile(f)" class="text-red-400 hover:text-red-300 text-sm px-2">✕</button>
          </div>
        </div>

        <!-- Options -->
        <div v-if="files.length" class="bg-gray-900 rounded-xl p-6 space-y-5">
          <h3 class="font-semibold text-gray-300 text-sm uppercase tracking-wide">Options</h3>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="text-xs text-gray-400 block mb-1.5">Format</label>
              <select v-model="format" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:border-purple-500 outline-none">
                <option value="nya">.nya (FEC Protected)</option>
                <option value="zip">.zip</option>
                <option value="tar.gz">.tar.gz</option>
                <option value="rar">.rar (Store)</option>
              </select>
            </div>
            <div>
              <label class="text-xs text-gray-400 block mb-1.5">Compression Level</label>
              <div class="flex items-center gap-3">
                <input type="range" v-model="level" min="1" max="19" class="flex-1 accent-purple-500">
                <span class="text-sm text-gray-400 w-6 text-right">{{ level }}</span>
              </div>
            </div>
            <div v-if="format==='nya'">
              <label class="text-xs text-gray-400 block mb-1.5">FEC Recovery %</label>
              <div class="flex items-center gap-3">
                <input type="range" v-model="fec" min="0" max="100" class="flex-1 accent-green-500">
                <span class="text-sm text-gray-400 w-10 text-right">{{ fec }}%</span>
              </div>
            </div>
            <div>
              <label class="text-xs text-gray-400 block mb-1.5">Password</label>
              <input type="password" v-model="password" placeholder="Optional"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:border-purple-500 outline-none">
            </div>
          </div>
          <div class="flex gap-6">
            <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
              <input type="checkbox" v-model="solid" class="accent-purple-500"> Solid Mode
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
              <input type="checkbox" v-model="sfx" class="accent-purple-500"> Self-Extracting
            </label>
          </div>

          <!-- Actions -->
          <div class="flex gap-3 pt-1">
            <button @click="pack" :disabled="packing"
              class="flex-1 bg-purple-600 hover:bg-purple-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ packing ? '⏳ Packing...' : '📦 Pack' }}
            </button>
            <button @click="extract" :disabled="extracting"
              class="flex-1 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 py-3 rounded-xl font-semibold transition text-sm">
              {{ extracting ? '⏳ Extracting...' : '📂 Extract' }}
            </button>
          </div>
        </div>

        <!-- Result -->
        <div v-if="result" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">
            {{ result.success ? '✅ Success' : '❌ Error' }}
          </p>
          <p class="text-sm text-gray-400 mt-1">{{ result.message }}</p>
        </div>
      </div>

      <!-- Repair -->
      <div v-if="currentView==='repair'" class="max-w-3xl mx-auto space-y-6">
        <div class="border-2 border-dashed border-gray-700 rounded-2xl p-16 text-center hover:border-green-600 transition-colors">
          <div class="text-6xl mb-4">🔧</div>
          <p class="text-xl font-medium text-gray-300">Drop a damaged .nya file</p>
          <p class="text-sm text-gray-500 mt-3">RaptorQ FEC can recover up to <span class="text-green-400 font-semibold">50% damage</span></p>
        </div>
        <button @click="repair" :disabled="repairing"
          class="w-full bg-green-600 hover:bg-green-500 disabled:opacity-50 py-4 rounded-xl font-semibold transition text-sm">
          {{ repairing ? '⏳ Repairing...' : '🔧 Repair Archive' }}
        </button>
        <div v-if="result && currentView==='repair'" class="rounded-xl p-4 border"
             :class="result.success ? 'bg-green-950/30 border-green-800' : 'bg-red-950/30 border-red-800'">
          <p :class="result.success ? 'text-green-400' : 'text-red-400'" class="font-medium text-sm">
            {{ result.success ? '✅ Repaired' : '❌ Failed' }}
          </p>
          <p class="text-sm text-gray-400 mt-1">{{ result.message }}</p>
        </div>
      </div>

      <!-- About -->
      <div v-if="currentView==='about'" class="max-w-3xl mx-auto space-y-6">
        <div class="bg-gray-900 rounded-xl p-8 text-center">
          <div class="text-6xl mb-4">🐱</div>
          <h2 class="text-2xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">
            NekoArc
          </h2>
          <p class="text-gray-400 mt-2">Next-generation archive manager with self-healing FEC</p>
        </div>
        <div class="bg-gray-900 rounded-xl p-6 grid grid-cols-2 gap-4 text-sm">
          <div class="space-y-3">
            <div><span class="text-gray-500">Core</span><br><span class="text-gray-300">Nyarc v0.6.0</span></div>
            <div><span class="text-gray-500">FEC</span><br><span class="text-gray-300">GoFEC (RaptorQ + LDPC)</span></div>
            <div><span class="text-gray-500">Compression</span><br><span class="text-gray-300">Zstd (1-19)</span></div>
          </div>
          <div class="space-y-3">
            <div><span class="text-gray-500">Hash</span><br><span class="text-gray-300">BLAKE3</span></div>
            <div><span class="text-gray-500">Encryption</span><br><span class="text-gray-300">AES-256-GCM</span></div>
            <div><span class="text-gray-500">Recovery</span><br><span class="text-green-400 font-semibold">Up to 50%</span></div>
          </div>
        </div>
        <div class="text-center text-xs text-gray-600">
          <p>© 2026 Nyarime • github.com/Nyarime/Nyarc</p>
        </div>
      </div>
    </main>

    <!-- Status Bar -->
    <footer class="bg-gray-900 border-t border-gray-800 px-6 py-2 flex justify-between text-xs text-gray-500">
      <span>NekoArc v0.1.0 — Nyarc Engine v0.6.0</span>
      <span>50% FEC Recovery • RaptorQ • BLAKE3 • AES-256-GCM</span>
    </footer>
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');

html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: 'Inter', system-ui, sans-serif;
  background: #030712;
}

/* Tailwind-like utilities via CDN */
</style>
