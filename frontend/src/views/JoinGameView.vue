<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { wsEndpoints } from '../config'
import { connectAndStoreSocket } from '../services/gameState'

const router = useRouter()
const joinRoomId = ref('')
const interactionError = ref('')

const handleJoinGame = async () => {
    if (!joinRoomId.value.trim()) return
    interactionError.value = ''
    const roomId = joinRoomId.value.trim()

    try {
        const wsUrl = wsEndpoints.joinRoom(roomId)
        await connectAndStoreSocket(wsUrl)
        router.push({ name: 'game' })
    } catch (err: any) {
        console.error('Failed to join game:', err)
        interactionError.value = 'Failed to join. Room may be full or does not exist.'
    }
}

const goBack = () => {
    router.push('/')
}
</script>

<template>
    <div class="min-h-screen bg-[#0F1115] text-[#E6E6E6] font-sans flex flex-col items-center justify-center p-6">
        <div class="max-w-md w-full">
            <button @click="goBack"
                class="mb-6 flex items-center text-[#A0A4AB] hover:text-[#E6E6E6] transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="19" y1="12" x2="5" y2="12"></line>
                    <polyline points="12 19 5 12 12 5"></polyline>
                </svg>
                Back to Home
            </button>

            <div
                class="bg-[#171A21] border border-[#242833] p-8 rounded-2xl shadow-xl hover:border-[#2ecc71]/50 transition-all duration-300 relative overflow-hidden group">
                <div
                    class="absolute inset-0 bg-gradient-to-br from-[#2ecc71]/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity">
                </div>

                <h2 class="text-2xl font-bold mb-6 flex items-center gap-2 relative z-10 text-[#E6E6E6]">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-7 h-7 text-[#2ecc71]" viewBox="0 0 24 24"
                        fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
                        stroke-linejoin="round">
                        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
                        <polyline points="10 17 15 12 10 7"></polyline>
                        <line x1="15" y1="12" x2="3" y2="12"></line>
                    </svg>
                    Join Game
                </h2>

                <div v-if="interactionError"
                    class="mb-4 p-3 bg-red-500/10 border border-red-500/50 rounded-lg text-red-500 text-sm font-medium relative z-10">
                    {{ interactionError }}
                </div>

                <div class="space-y-6 relative z-10">
                    <div>
                        <label class="block text-xs font-semibold text-[#A0A4AB] uppercase mb-1">Room Code</label>
                        <input type="text" :value="joinRoomId"
                            @input="joinRoomId = ($event.target as HTMLInputElement).value.toUpperCase()"
                            placeholder="Enter Code (e.g., A1B2)"
                            class="w-full bg-[#0F1115] border border-[#242833] rounded px-3 py-4 text-[#E6E6E6] focus:border-[#2ecc71] outline-none transition-colors font-mono text-xl tracking-wider text-center"
                            @keyup.enter="handleJoinGame" />
                    </div>

                    <button @click="handleJoinGame"
                        class="w-full py-3.5 bg-[#2ecc71] hover:bg-[#27ae60] text-white font-bold rounded-lg shadow-lg hover:shadow-[#2ecc71]/20 transition-all active:scale-[0.98]"
                        :disabled="!joinRoomId">
                        Join Room
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>
