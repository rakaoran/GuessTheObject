<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { wsEndpoints } from '../config'
import Dende from '@rakaoran/dende'
import { ServerPacket, ClientPacket, DrawingData } from '../proto/serialization'
import { activeSocket, bufferedMessages, clearGameState } from '../services/gameState'

// --- Game State ---
const route = useRoute()
const router = useRouter()
const dendeContainer = ref<HTMLElement | null>(null)
const socket = ref<WebSocket | null>(null)
const isConnected = ref(false)
const statusMessage = ref('Connecting...')
const chatBox = ref<HTMLElement | null>(null)
const chatInput = ref('')

interface Player {
  username: string
  score: number
  guessed: boolean
}

const players = ref<Player[]>([])
const chatMessages = ref<any[]>([])
const wordChoices = ref<string[]>([])
const showTurnSummary = ref(false)
const showLeaderboard = ref(false)
const turnSummaryWord = ref('')
const turnSummaryDeltas = ref<Array<{ username: string; scoreDelta: number }>>([])
const hasGameStarted = ref(false)
const isMyTurnToDraw = ref(false)
const myDrawingWord = ref('')
const gameRoomId = ref('')
const currentDrawer = ref('')
const isHost = ref(false)
const myUsername = ref('')
const isCreator = ref(false)
const currentRound = ref(1)
const isGameFinished = ref(false)

const currentColor = ref('#000000')
const currentSize = ref(5)
const currentMode = ref<'drawing' | 'filling'>('drawing')

// Timer State
const timeRemaining = ref(0)
const serverTimeOffset = ref(Number.MAX_SAFE_INTEGER)
const isTimeSynced = ref(false)
const targetServerTime = ref(0)
const choosingDuration = ref(15)
const drawingDuration = ref(60)
let timerInterval: number | null = null

let dende: Dende | null = null

// --- Helper Functions ---
const addPlayerIfNeeded = (username: string) => {
  if (username && !players.value.find(p => p.username === username)) {
    players.value.push({ username, score: 0, guessed: false })
  }
}

const scrollToChatBottom = () => {
  nextTick(() => {
    if (chatBox.value) {
      chatBox.value.scrollTop = chatBox.value.scrollHeight
    }
  })
}

const pushSystemMessage = (content: string) => {
  chatMessages.value.push({
    from: 'System',
    content: content,
    isSystem: true
  })
  scrollToChatBottom()
}

const syncTime = (serverTs: number) => {
  if (!serverTs) return
  // serverTimeOffset = localTime - serverTime
  // We want the MINIMUM offset (representing smallest latency in the path)
  const currentOffset = Date.now() - serverTs
  if (!isTimeSynced.value || currentOffset < serverTimeOffset.value) {
    serverTimeOffset.value = currentOffset
    isTimeSynced.value = true
  }
}

const updateTimer = () => {
  if (targetServerTime.value <= 0) {
    timeRemaining.value = 0
    return
  }
  const serverNow = Date.now() - serverTimeOffset.value
  const diff = targetServerTime.value - serverNow
  timeRemaining.value = Math.max(0, Math.ceil(diff / 1000))
}


const connect = async () => {

  const handleOpen = () => {

    if (activeSocket.value) {
      console.log('Already connected to a game socket 🚀')
    }

    console.log('Connected to Room! 🚀')
    statusMessage.value = 'Connected'

    const hasCreationParams = route.query.maxPlayers !== undefined || route.query.private !== undefined
    console.log('Creation params detected:', hasCreationParams, route.query)

    isCreator.value = hasCreationParams
    if (isCreator.value) {
      isHost.value = true
      statusMessage.value = 'Creating game...'
    }
    isConnected.value = true
  }

  if (activeSocket.value && activeSocket.value.readyState === WebSocket.OPEN) {
    console.log('Using pre-connected socket')
    socket.value = activeSocket.value
    handleOpen()
  } else {
    let wsUrl = ''

    const params = new URLSearchParams({
      private: 'false',
      maxPlayers: '8',
      roundsCount: '3',
      wordsCount: '3',
      choosingWordDuration: '15',
      drawingDuration: '60'
    })
    wsUrl = `${wsEndpoints.createGame}?${params.toString()}`

    socket.value = new WebSocket(wsUrl)
    socket.value.binaryType = 'arraybuffer'
    socket.value.onopen = handleOpen
  }

  socket.value.onmessage = async (event) => {
    try {
      const buffer = event.data as ArrayBuffer
      if (buffer.byteLength === 0) return

      const data = new Uint8Array(buffer)
      const packet = ServerPacket.decode(data)

      handleServerPacket(packet)
    } catch (err) {
      console.error('Error processing message:', err)
    }
  }

  socket.value.onclose = (event: CloseEvent) => {
    isConnected.value = false
    console.log('Socket closed:', event.code, event.reason)

    if (isGameFinished.value) {
      statusMessage.value = 'Game Over'
      return
    }

    if (hasGameStarted.value && players.value.length <= 1) {
      statusMessage.value = 'Game Ended'
      pushSystemMessage('All other players have left. Game over.')
      alert('All other players have left the game.')
      router.push('/')
      return
    }

    if (event.reason === 'room-not-found') {
      statusMessage.value = '❌ Room not found!'
      pushSystemMessage('Error: That room doesn\'t exist. 🤷‍♂️')
    } else if (event.reason === 'room-full') {
      statusMessage.value = '❌ Room is full!'
      pushSystemMessage('Error: That room is already full. 🚫')
    } else if (event.code === 1006) {
      statusMessage.value = 'Disconnected'
      pushSystemMessage('Connection lost.')
    } else {
      statusMessage.value = 'Disconnected'
    }
  }

  if (bufferedMessages.value.length > 0) {
    console.log('Replaying', bufferedMessages.value.length, 'buffered messages')
    bufferedMessages.value.forEach(evt => {
      if (socket.value && socket.value.onmessage) {
        socket.value.onmessage(evt)
      }
    })
    clearGameState()
  }
}

// --- Server   Packet Handler ---
const handleServerPacket = (packet: ServerPacket) => {
  // console.log('Received packet keys:', Object.keys(packet)) 

  if (packet.serverTimestamp) {
    syncTime(packet.serverTimestamp)
  }

  // Explicit check for leaderboard existence (since it might be an empty object)
  if (packet.leaderboard !== undefined) {
    console.log('Leaderboard packet detected:', packet.leaderboard)
    handleLeaderboard()
  }

  if (packet.initialRoomSnapshot) {
    handleInitialRoomSnapshot(packet.initialRoomSnapshot)
  } else if (packet.playerJoined) {
    handlePlayerJoined(packet.playerJoined)
  } else if (packet.playerLeft) {
    handlePlayerLeft(packet.playerLeft)
  } else if (packet.gameStarted) {
    handleGameStarted()
  } else if (packet.playerIsChoosingWord) {
    handlePlayerIsChoosingWord(packet.playerIsChoosingWord, packet.serverTimestamp)
  } else if (packet.pleaseChooseAWord) {
    handlePleaseChooseAWord(packet.pleaseChooseAWord, packet.serverTimestamp)
  } else if (packet.playerIsDrawing) {
    handlePlayerIsDrawing(packet.playerIsDrawing, packet.serverTimestamp)
  } else if (packet.yourTurnToDraw) {
    handleYourTurnToDraw(packet.yourTurnToDraw, packet.serverTimestamp)
  } else if (packet.drawingData) {
    handleDrawingData(packet.drawingData)
  } else if (packet.playerGuessedTheWord) {
    handlePlayerGuessedTheWord(packet.playerGuessedTheWord)
  } else if (packet.turnSummary) {
    handleTurnSummary(packet.turnSummary)
  } else if (packet.playerMessage) {
    handlePlayerMessage(packet.playerMessage)
  } else if (packet.roundUpdate) {
    handleRoundUpdate(packet.roundUpdate)
  }
}

const handleInitialRoomSnapshot = (snapshot: ServerPacket['initialRoomSnapshot']) => {
  if (!snapshot) return

  // Set room ID
  gameRoomId.value = snapshot.roomId

  // Set players
  players.value = snapshot.playersStates.map(ps => ({
    username: ps.username,
    score: Number(ps.score),
    guessed: ps.isGuesser
  }))

  // Get current username
  const storedUsername = localStorage.getItem('username')
  myUsername.value = storedUsername || ''

  if (myUsername.value && !players.value.find(p => p.username === myUsername.value)) {
    players.value.push({
      username: myUsername.value,
      score: 0,
      guessed: false
    })
  }

  // Determine if this user is the host
  if (isCreator.value) {
    isHost.value = true
  } else {
    // If joining, check if first player matches me
    isHost.value = players.value.length > 0 && players.value[0].username === myUsername.value
  }

  // Determine Game State (Wait vs Play)
  // If round > 0, the game has already started, so show the canvas.
  if (snapshot.currentRound > 0) {
    currentRound.value = snapshot.currentRound
    hasGameStarted.value = true
    statusMessage.value = 'Game in progress...'
    // If joining mid-game, ensure we don't show turn summary immediately unless needed
    showTurnSummary.value = false
  } else {
    // Round 0 implies waiting room
    hasGameStarted.value = false
    statusMessage.value = 'Waiting for players...'
  }

  // Set durations and sync timer
  if (snapshot.choosingWordDuration) choosingDuration.value = snapshot.choosingWordDuration
  if (snapshot.drawingDuration) drawingDuration.value = snapshot.drawingDuration

  if (snapshot.nextTick) {
    targetServerTime.value = snapshot.nextTick
  }

  // Set current drawer
  currentDrawer.value = snapshot.currentDrawer

  // Replay drawing history
  if (snapshot.drawingHistory && dende) {
    for (const historyBytes of snapshot.drawingHistory) {
      dende.putPart(new Uint8Array(historyBytes))
    }
  }

  console.log('Synced initial room snapshot:', snapshot)
}

const handlePlayerJoined = (data: ServerPacket['playerJoined']) => {
  if (!data) return
  addPlayerIfNeeded(data.username)
  pushSystemMessage(`${data.username} joined!`)
}

const handlePlayerLeft = (data: ServerPacket['playerLeft']) => {
  if (!data) return
  players.value = players.value.filter(p => p.username !== data.username)
  pushSystemMessage(`${data.username} left.`)
}

const handleGameStarted = () => {
  hasGameStarted.value = true
  showTurnSummary.value = false
  statusMessage.value = 'Game started!'
  pushSystemMessage('Game started! 🚀')
}

const handlePlayerIsChoosingWord = (data: ServerPacket['playerIsChoosingWord'], serverTs?: number) => {
  if (!data) return
  showTurnSummary.value = false
  showLeaderboard.value = false
  addPlayerIfNeeded(data.username)
  currentDrawer.value = data.username
  statusMessage.value = `${data.username} is choosing...`
  isMyTurnToDraw.value = false
  dende?.disableDrawing()
  dende?.clear()
  const baseTime = serverTs || (Date.now() - serverTimeOffset.value)
  targetServerTime.value = baseTime + (choosingDuration.value * 1000)
  pushSystemMessage(`${data.username} is choosing a word...`)
}

const handlePleaseChooseAWord = (data: ServerPacket['pleaseChooseAWord'], serverTs?: number) => {
  if (!data) return
  showTurnSummary.value = false
  showLeaderboard.value = false
  statusMessage.value = 'Your turn! Choose a word:'
  wordChoices.value = data.words
  isMyTurnToDraw.value = true
  currentDrawer.value = myUsername.value
  const baseTime = serverTs || (Date.now() - serverTimeOffset.value)
  targetServerTime.value = baseTime + (choosingDuration.value * 1000)
}

const handlePlayerIsDrawing = (data: ServerPacket['playerIsDrawing'], serverTs?: number) => {
  if (!data) return
  showTurnSummary.value = false
  showLeaderboard.value = false
  addPlayerIfNeeded(data.username)
  currentDrawer.value = data.username
  statusMessage.value = `${data.username} is drawing!`
  wordChoices.value = []
  dende?.clear()
  dende?.disableDrawing()
  dende?.disableDrawing()
  const baseTime = serverTs || (Date.now() - serverTimeOffset.value)
  targetServerTime.value = baseTime + (drawingDuration.value * 1000)
  pushSystemMessage(`${data.username} is drawing!`)
}

const handleYourTurnToDraw = (data: ServerPacket['yourTurnToDraw'], serverTs?: number) => {
  if (!data) return
  showTurnSummary.value = false
  showLeaderboard.value = false
  myDrawingWord.value = data.word
  statusMessage.value = `Drawing: ${data.word}`
  wordChoices.value = []
  dende?.clear()
  dende?.enableDrawing()
  updateSettings()
  pushSystemMessage(`Your turn to draw: ${data.word}`)
  currentDrawer.value = myUsername.value
  const baseTime = serverTs || (Date.now() - serverTimeOffset.value)
  targetServerTime.value = baseTime + (drawingDuration.value * 1000)
}

const handleDrawingData = (data: DrawingData | undefined) => {
  if (!data || isMyTurnToDraw.value || !dende) return
  dende.putPart(new Uint8Array(data.data))
}

const handlePlayerGuessedTheWord = (data: ServerPacket['playerGuessedTheWord']) => {
  if (!data) return
  addPlayerIfNeeded(data.username)
  const p = players.value.find(p => p.username === data.username)
  if (p) p.guessed = true
  pushSystemMessage(`${data.username} guessed it! 🎉`)
}

const handleTurnSummary = (summary: ServerPacket['turnSummary']) => {
  if (!summary) return

  turnSummaryWord.value = summary.wordReveal
  turnSummaryDeltas.value = summary.deltas.map(d => ({
    username: d.username,
    scoreDelta: Number(d.scoreDelta)
  }))

  // Update player scores
  for (const delta of summary.deltas) {
    const player = players.value.find(p => p.username === delta.username)
    if (player) {
      player.score += Number(delta.scoreDelta)
      player.guessed = false
    }
  }

  // Sort by score
  players.value.sort((a, b) => b.score - a.score)

  showTurnSummary.value = true
  showLeaderboard.value = false
  statusMessage.value = 'Turn over!'
  myDrawingWord.value = ''
  dende?.reset()
  targetServerTime.value = 0
}

const handleLeaderboard = () => {
  showTurnSummary.value = false
  showLeaderboard.value = true
  isGameFinished.value = true
  targetServerTime.value = 0
  statusMessage.value = 'Game Over! 🏆'
  pushSystemMessage('Game Over! Check the leaderboard! 🏆')

  // Ensure players are sorted by score
  players.value.sort((a, b) => b.score - a.score)
}

const handlePlayerMessage = (msg: ServerPacket['playerMessage']) => {
  if (!msg) return
  addPlayerIfNeeded(msg.from)
  chatMessages.value.push({ from: msg.from, content: msg.message, isSystem: false })
  scrollToChatBottom()
}

const handleRoundUpdate = (update: ServerPacket['roundUpdate']) => {
  if (!update) return
  currentRound.value = update.roundNumber
  pushSystemMessage(`Round ${update.roundNumber} starting!`)
}

// --- Sending Functions ---
const sendPart = (bytes: Uint8Array) => {
  if (!socket.value || socket.value.readyState !== WebSocket.OPEN) return

  const clientPacket = ClientPacket.create({
    drawingData: { data: bytes }
  })

  const encoded = ClientPacket.encode(clientPacket).finish()
  socket.value.send(encoded)
}

const sendWordChoice = (index: number) => {
  if (!socket.value) return

  const clientPacket = ClientPacket.create({
    wordChoice: { choice: index }
  })

  const encoded = ClientPacket.encode(clientPacket).finish()
  socket.value.send(encoded)
  wordChoices.value = []
}

const sendChatMessage = () => {
  if (!socket.value || !chatInput.value.trim()) return

  const clientPacket = ClientPacket.create({
    playerMessage: { message: chatInput.value }
  })

  const encoded = ClientPacket.encode(clientPacket).finish()
  socket.value.send(encoded)

  chatMessages.value.push({ from: 'You', content: chatInput.value, isSystem: false })
  chatInput.value = ''
  scrollToChatBottom()
}

const sendStartGame = () => {
  if (!socket.value || !isHost.value || players.value.length < 2) return

  const clientPacket = ClientPacket.create({
    startGame: {}
  })

  const encoded = ClientPacket.encode(clientPacket).finish()
  socket.value.send(encoded)
}

const copyInviteLink = () => {
  if (!gameRoomId.value) return
  // Just copy the ID now
  const textToCopy = gameRoomId.value
  navigator.clipboard.writeText(textToCopy).then(() => {
    pushSystemMessage('Room ID copied! 📋')
  }).catch(err => {
    console.error('Failed to copy: ', err)
    pushSystemMessage('Failed to copy ID.')
  })
}

const updateSettings = () => {
  if (!dende) return
  const hex = currentColor.value
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)

  dende.setLineColorRGBA(r, g, b, 1)
  dende.setLineWidth(currentSize.value)
  dende.setDrawingMode(currentMode.value)
}

const undo = () => dende?.undo()
const redo = () => dende?.redo()
const clear = () => dende?.clear()

const goHome = () => {
  router.push('/')
}

onMounted(() => {
  dende = new Dende(800, 600)
  dende.setFPS(60)

  if (dendeContainer.value) {
    dendeContainer.value.appendChild(dende.getHTMLElement())
  }

  dende.addPartListener((part) => {
    if (isMyTurnToDraw.value) {
      sendPart(part)
    }
  })

  dende.disableDrawing()
  updateSettings()
  connect()

  timerInterval = setInterval(updateTimer, 100) as unknown as number
})

onUnmounted(() => {
  socket.value?.close()
  if (timerInterval) clearInterval(timerInterval)
})
</script>

<template>
  <div
    class="min-h-screen flex flex-col items-center justify-center bg-[#0F1115] text-[#E6E6E6] p-4 font-sans selection:bg-[#4C8DFF] selection:text-white">

    <div class="w-full max-w-[1400px] mb-4">
      <div class="flex items-center justify-between p-3 bg-[#171A21] border border-[#242833] rounded shadow-sm">
        <div class="flex items-center gap-3">
          <div class="w-2.5 h-2.5 rounded-full" :class="isConnected ? 'bg-[#4C8DFF]' : 'bg-red-500'"></div>
          <span class="text-sm font-medium text-[#A0A4AB]">Status: <span class="text-[#E6E6E6]">{{ statusMessage
          }}</span></span>
        </div>
        <div class="flex items-center gap-4">
          <div v-if="timeRemaining > 0 && hasGameStarted"
            class="flex items-center gap-2 bg-[#242833] px-3 py-1 rounded">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-[#A0A4AB]" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
            <span class="font-mono font-bold text-[#E6E6E6]"
              :class="{ 'text-red-400': timeRemaining <= 10, 'text-[#4C8DFF]': timeRemaining > 10 }">
              {{ Math.floor(timeRemaining / 60) }}:{{ (timeRemaining % 60).toString().padStart(2, '0') }}
            </span>
          </div>
          <div v-if="gameRoomId" class="text-sm text-[#A0A4AB]">
            Room: <span class="text-[#E6E6E6] font-mono">{{ gameRoomId }}</span>
          </div>
          <div v-if="hasGameStarted" class="text-sm text-[#A0A4AB]">
            Round: <span class="text-[#E6E6E6] font-mono">{{ currentRound }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-col lg:flex-row w-full max-w-[1400px] gap-6 h-[85vh]">

      <!-- GAME UI (Sidebar) -->
      <div v-if="hasGameStarted"
        class="w-full lg:w-64 bg-[#171A21] border border-[#242833] flex flex-col shrink-0 rounded overflow-hidden">
        <div class="p-4 border-b border-[#242833] bg-[#171A21]">
          <h3 class="text-sm font-semibold text-[#E6E6E6]">Players</h3>
          <p class="text-xs text-[#A0A4AB] mt-1">{{ players.length }} connected</p>
        </div>

        <div
          class="p-3 space-y-2 overflow-y-auto grow scrollbar-thin scrollbar-thumb-[#242833] scrollbar-track-transparent">

          <ul class="space-y-1">
            <li v-for="player in players" :key="player.username"
              class="flex justify-between items-center p-2 rounded border border-transparent transition-colors"
              :class="player.guessed ? 'bg-[#4C8DFF]/10 border-[#4C8DFF]/30' : 'hover:bg-[#242833]/50'">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium" :class="player.guessed ? 'text-[#4C8DFF]' : 'text-[#E6E6E6]'">
                  {{ player.username }}
                </span>
                <svg v-if="player.guessed" xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 text-[#4C8DFF]"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"
                  stroke-linejoin="round">
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
                <svg v-if="player.username === currentDrawer" xmlns="http://www.w3.org/2000/svg"
                  class="w-3.5 h-3.5 text-[#f39c12]" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"></path>
                </svg>
              </div>
              <span class="text-xs font-mono text-[#A0A4AB]">{{ player.score }}</span>
            </li>
          </ul>
        </div>
      </div>

      <!-- MAIN AREA -->
      <div class="grow flex flex-col min-w-0">

        <!-- CANVAS / GAME BOARD -->
        <div v-show="hasGameStarted"
          class="relative grow bg-[#121419] border border-[#242833] rounded-t flex flex-col items-center justify-center overflow-hidden">

          <div ref="dendeContainer" class="cursor-crosshair shadow-lg bg-white"
            style="touch-action: none; max-width: 100%; max-height: 100%;">
          </div>

          <!-- Overlays (Turn Summary, Leaderboard, Word Choice, etc.) -->
          <div v-if="showTurnSummary"
            class="absolute inset-0 bg-[#0F1115]/90 backdrop-blur-sm flex items-center justify-center z-50">
            <div class="bg-[#171A21] border border-[#242833] rounded-lg p-6 w-full max-w-sm shadow-xl">
              <h2 class="text-lg font-semibold text-[#E6E6E6] mb-2 text-center">The word was:</h2>
              <p class="text-2xl font-bold text-[#4C8DFF] text-center mb-4">{{ turnSummaryWord }}</p>
              <h3 class="text-sm font-semibold text-[#A0A4AB] mb-3">Score Changes:</h3>
              <ul class="space-y-2 mb-4">
                <li v-for="delta in turnSummaryDeltas" :key="delta.username"
                  class="flex justify-between p-2 border-b border-[#242833] text-sm">
                  <span class="text-[#E6E6E6]">{{ delta.username }}</span>
                  <span class="font-bold font-mono" :class="delta.scoreDelta > 0 ? 'text-[#4C8DFF]' : 'text-[#A0A4AB]'">
                    <span v-if="delta.scoreDelta > 0">+</span>{{ delta.scoreDelta }}
                  </span>
                </li>
              </ul>
            </div>
          </div>

          <div v-if="showLeaderboard"
            class="absolute inset-0 bg-[#0F1115]/95 flex items-center justify-center z-50 p-4">
            <div class="flex flex-col items-center w-full max-w-4xl">
              <h2
                class="text-4xl md:text-5xl font-black mb-8 text-transparent bg-clip-text bg-gradient-to-r from-[#FFD700] via-[#FDB931] to-[#FFD700] tracking-wider drop-shadow-sm">
                LEADERBOARD
              </h2>

              <!-- Podium Layout -->
              <div class="flex items-end justify-center gap-4 mb-12 w-full min-h-[300px]">

                <!-- 2nd Place -->
                <div v-if="players.length > 1" class="flex flex-col items-center w-1/3 max-w-[200px] animate-fade-in-up"
                  style="animation-delay: 0.2s">
                  <div class="mb-2 text-center">
                    <span class="block text-xl font-bold text-[#E6E6E6] truncate w-full">{{ players[1].username
                      }}</span>
                    <span class="block text-sm text-[#A0A4AB] font-mono">{{ players[1].score }} pts</span>
                  </div>
                  <div
                    class="w-full h-32 bg-[#C0C0C0] rounded-t-xl relative border-t-4 border-l-2 border-r-2 border-[#E6E6E6]/20 shadow-[0_0_30px_rgba(192,192,192,0.2)] flex items-end justify-center pb-4">
                    <span class="text-4xl font-black text-[#171A21] opacity-50">2</span>
                  </div>
                </div>

                <!-- 1st Place -->
                <div v-if="players.length > 0"
                  class="flex flex-col items-center w-1/3 max-w-[220px] z-10 animate-fade-in-up">
                  <div class="mb-2 text-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8 text-[#FFD700] mx-auto mb-1 animate-bounce"
                      viewBox="0 0 24 24" fill="currentColor">
                      <path
                        d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z">
                      </path>
                    </svg>
                    <span class="block text-2xl font-bold text-[#FFD700] truncate w-full">{{ players[0].username
                      }}</span>
                    <span class="block text-base text-[#FFD700]/80 font-mono font-bold">{{ players[0].score }}
                      pts</span>
                  </div>
                  <div
                    class="w-full h-48 bg-gradient-to-br from-[#FFD700] to-[#FDB931] rounded-t-xl relative border-t-4 border-l-2 border-r-2 border-[#FFFFE0]/30 shadow-[0_0_50px_rgba(255,215,0,0.4)] flex items-end justify-center pb-4">
                    <span class="text-6xl font-black text-[#8B4513] opacity-40">1</span>
                  </div>
                </div>

                <!-- 3rd Place -->
                <div v-if="players.length > 2" class="flex flex-col items-center w-1/3 max-w-[200px] animate-fade-in-up"
                  style="animation-delay: 0.4s">
                  <div class="mb-2 text-center">
                    <span class="block text-xl font-bold text-[#E6E6E6] truncate w-full">{{ players[2].username
                      }}</span>
                    <span class="block text-sm text-[#A0A4AB] font-mono">{{ players[2].score }} pts</span>
                  </div>
                  <div
                    class="w-full h-24 bg-[#CD7F32] rounded-t-xl relative border-t-4 border-l-2 border-r-2 border-[#E6E6E6]/20 shadow-[0_0_30px_rgba(205,127,50,0.2)] flex items-end justify-center pb-4">
                    <span class="text-4xl font-black text-[#3E1C00] opacity-30">3</span>
                  </div>
                </div>
              </div>

              <!-- Other Players List -->
              <div class="w-full max-w-lg bg-[#171A21] border border-[#242833] rounded-xl overflow-hidden mb-8"
                v-if="players.length > 3">
                <div v-for="(player, index) in players.slice(3)" :key="player.username"
                  class="flex justify-between items-center p-3 border-b border-[#242833] last:border-0 hover:bg-[#242833]/50 transition-colors">
                  <div class="flex items-center gap-3">
                    <span class="font-mono text-[#A0A4AB] text-sm w-6">#{{ index + 4 }}</span>
                    <span class="font-medium text-[#E6E6E6]">{{ player.username }}</span>
                  </div>
                  <span class="font-mono text-sm text-[#A0A4AB]">{{ player.score }} pts</span>
                </div>
              </div>

              <button @click="goHome"
                class="bg-[#4C8DFF] hover:bg-[#3b7cdb] text-white font-bold py-3 px-8 rounded-full shadow-lg hover:shadow-[#4C8DFF]/25 transform hover:-translate-y-0.5 transition-all">
                Return to Main Menu
              </button>
            </div>
          </div>

          <div v-if="wordChoices.length > 0"
            class="absolute inset-0 bg-[#0F1115]/80 backdrop-blur-sm flex items-center justify-center z-50">
            <div class="bg-[#171A21] p-6 border border-[#242833] rounded-lg shadow-xl max-w-lg w-full">
              <h2 class="text-lg font-semibold text-center mb-1 text-[#E6E6E6]">Your Turn</h2>
              <p class="text-center text-[#A0A4AB] text-sm mb-6">Choose a word to draw</p>

              <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <button v-for="(word, index) in wordChoices" :key="word" @click="sendWordChoice(index)"
                  class="px-4 py-4 bg-[#0F1115] border border-[#242833] rounded hover:border-[#4C8DFF] hover:text-[#4C8DFF] transition-all text-sm font-medium text-[#E6E6E6]">
                  {{ word }}
                </button>
              </div>
            </div>
          </div>

          <div v-if="isMyTurnToDraw && myDrawingWord"
            class="absolute top-4 left-1/2 -translate-x-1/2 z-30 pointer-events-none">
            <div class="bg-[#4C8DFF] text-white px-4 py-1.5 rounded-full text-xs font-semibold shadow-lg">
              Drawing: {{ myDrawingWord }}
            </div>
          </div>
        </div>

        <!-- TOOLBAR (Only during game) -->
        <div v-if="hasGameStarted"
          class="bg-[#171A21] border-x border-b border-[#242833] p-3 flex items-center gap-4 rounded-b relative z-20"
          :class="{ 'opacity-50 pointer-events-none': !isMyTurnToDraw }">

          <div class="relative w-8 h-8 shrink-0 rounded overflow-hidden border border-[#242833]">
            <input type="color" v-model="currentColor" @change="updateSettings"
              class="w-[150%] h-[150%] absolute -top-1/4 -left-1/4 cursor-pointer p-0 border-0">
          </div>

          <div class="flex flex-col grow gap-1 max-w-[200px]">
            <div class="flex justify-between items-center text-[10px] uppercase font-medium text-[#A0A4AB]">
              <span>Size</span>
              <span>{{ currentSize }}px</span>
            </div>
            <input type="range" v-model.number="currentSize" @input="updateSettings" min="1" max="20"
              class="w-full h-1 bg-[#242833] rounded-lg appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:bg-[#4C8DFF] [&::-webkit-slider-thumb]:rounded-full">
          </div>

          <div class="w-px h-8 bg-[#242833] mx-2"></div>

          <div class="flex gap-1">
            <button @click="() => { currentMode = 'drawing'; updateSettings() }" class="p-2 rounded transition-colors"
              :class="currentMode === 'drawing' ? 'bg-[#242833] text-[#4C8DFF]' : 'text-[#A0A4AB] hover:bg-[#242833]/50'">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 19l7-7 3 3-7 7-3-3z"></path>
                <path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z"></path>
                <path d="M2 2l7.586 7.586"></path>
                <circle cx="11" cy="11" r="2"></circle>
              </svg>
            </button>
            <button @click="() => { currentMode = 'filling'; updateSettings() }" class="p-2 rounded transition-colors"
              :class="currentMode === 'filling' ? 'bg-[#242833] text-[#4C8DFF]' : 'text-[#A0A4AB] hover:bg-[#242833]/50'">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M19 11l-8-8-8.6 8.6a2 2 0 0 0 0 2.8l5.2 5.2c.8.8 2 .8 2.8 0L19 11z"></path>
                <path d="M5 2l5 5"></path>
                <path d="M2 13h15"></path>
                <path d="M22 20a2 2 0 1 1-4 0c0-1.6 1.7-2.4 2-4 .3 1.6 2 2.4 2 4z"></path>
              </svg>
            </button>
          </div>

          <div class="w-px h-8 bg-[#242833] mx-2"></div>

          <div class="flex gap-1">
            <button @click="undo"
              class="p-2 text-[#A0A4AB] hover:text-[#E6E6E6] hover:bg-[#242833]/50 rounded transition-colors"
              title="Undo">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 7v6h6"></path>
                <path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13"></path>
              </svg>
            </button>
            <button @click="redo"
              class="p-2 text-[#A0A4AB] hover:text-[#E6E6E6] hover:bg-[#242833]/50 rounded transition-colors"
              title="Redo">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 7v6h-6"></path>
                <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 2.7"></path>
              </svg>
            </button>
            <button @click="clear" class="p-2 text-red-400 hover:bg-red-500/10 rounded transition-colors ml-2"
              title="Clear Canvas">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                <line x1="10" y1="11" x2="10" y2="17"></line>
                <line x1="14" y1="11" x2="14" y2="17"></line>
              </svg>
            </button>
          </div>
        </div>

        <!-- ROOM UI (When game hasn't started) -->
        <div v-if="!hasGameStarted"
          class="grow bg-[#121419] border border-[#242833] rounded flex flex-col items-center justify-center p-8 relative overflow-hidden">

          <div
            class="absolute inset-0 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-stops))] from-[#4C8DFF]/10 via-[#0F1115] to-[#0F1115]">
          </div>

          <div class="relative z-10 w-full max-w-2xl flex flex-col items-center">

            <div class="bg-[#171A21] border border-[#242833] rounded-xl p-8 w-full shadow-2xl">
              <div class="text-center mb-8">
                <h2 class="text-3xl font-bold text-[#E6E6E6] mb-2">Room</h2>
                <p class="text-[#A0A4AB]">Waiting for players to join...</p>
              </div>

              <div v-if="gameRoomId" class="flex flex-col items-center mb-8">
                <label class="text-xs uppercase font-semibold text-[#4C8DFF] mb-2">Room Code</label>
                <div
                  class="flex items-center gap-2 bg-[#0F1115] border border-[#242833] rounded-lg p-2 pl-4 w-full max-w-sm">
                  <code class="text-xl font-mono text-[#E6E6E6] grow text-center">{{ gameRoomId }}</code>
                  <button @click="copyInviteLink"
                    class="p-2 hover:bg-[#242833] rounded text-[#A0A4AB] hover:text-[#E6E6E6] transition-colors"
                    title="Copy Link">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none"
                      stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    </svg>
                  </button>
                </div>
              </div>

              <div class="mb-8">
                <h3 class="text-sm font-semibold text-[#E6E6E6] mb-4 flex items-center justify-between">
                  Players
                  <span class="bg-[#4C8DFF]/10 text-[#4C8DFF] px-2 py-0.5 rounded text-xs">{{ players.length }}</span>
                </h3>
                <div class="grid grid-cols-2 md:grid-cols-3 md:grid-cols-4 gap-3">
                  <div v-for="player in players" :key="player.username"
                    class="bg-[#0F1115] border border-[#242833] p-3 rounded flex flex-col items-center gap-2 animate-in fade-in zoom-in duration-300">
                    <div
                      class="w-10 h-10 rounded-full bg-gradient-to-br from-[#4C8DFF] to-[#2ecc71] flex items-center justify-center text-white font-bold shadow-lg">
                      {{ player.username.charAt(0).toUpperCase() }}
                    </div>
                    <span class="text-sm text-[#E6E6E6] font-medium truncate max-w-full">{{ player.username }}</span>
                    <span v-if="player.username === players[0]?.username"
                      class="text-[10px] text-[#f39c12] uppercase tracking-wider font-bold">Host</span>
                  </div>
                </div>
              </div>

              <div class="flex flex-col gap-3">
                <button v-if="isHost" @click="sendStartGame" :disabled="players.length < 2"
                  class="w-full py-4 font-bold rounded-lg shadow-lg transition-all"
                  :class="players.length < 2 ? 'bg-[#242833] text-[#A0A4AB] cursor-not-allowed border border-[#242833]' : 'bg-[#4C8DFF] hover:bg-[#3b7cdb] text-white hover:shadow-[#4C8DFF]/20 active:scale-[0.98]'">
                  {{ players.length < 2 ? 'Waiting for players...' : 'Start Game' }} </button>
                    <div v-else
                      class="w-full py-4 bg-[#242833] text-[#A0A4AB] font-medium rounded-lg text-center cursor-not-allowed">
                      Waiting for host to start...
                    </div>
              </div>

            </div>

          </div>
        </div>

      </div>

      <!-- CHAT (Always visible) -->
      <div
        class="w-full lg:w-72 bg-[#171A21] border border-[#242833] flex flex-col shrink-0 rounded overflow-hidden h-64 lg:h-auto">
        <div class="p-4 border-b border-[#242833] bg-[#171A21]">
          <h3 class="text-sm font-semibold text-[#E6E6E6]">Chat</h3>
        </div>

        <div ref="chatBox"
          class="grow overflow-y-auto p-3 space-y-2 text-sm scrollbar-thin scrollbar-thumb-[#242833] scrollbar-track-transparent">
          <div v-for="(msg, index) in chatMessages" :key="index" class="wrap-break-word">

            <div v-if="msg.isSystem" class="text-center py-0.5">
              <span class="inline-block bg-[#242833]/50 text-[#A0A4AB] px-2 py-0.5 rounded text-[10px]">
                {{ msg.content }}
              </span>
            </div>

            <div v-else>
              <span class="font-bold" :class="msg.from === 'You' ? 'text-[#4C8DFF]' : 'text-[#A0A4AB]'">
                {{ msg.from }}:
              </span>
              <span class="text-[#E6E6E6] ml-1">{{ msg.content }}</span>
            </div>
          </div>
        </div>

        <form @submit.prevent="sendChatMessage" class="p-3 bg-[#171A21] border-t border-[#242833] flex gap-2">
          <input v-model="chatInput" type="text" placeholder="Type a guess..."
            class="grow min-w-0 bg-[#0F1115] border border-[#242833] focus:border-[#4C8DFF] text-[#E6E6E6] text-sm px-3 py-2 rounded outline-none placeholder-[#A0A4AB] transition-colors" />
        </form>
      </div>

    </div>
  </div>
</template>