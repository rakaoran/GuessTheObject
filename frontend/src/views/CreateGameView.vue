<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { wsEndpoints } from '../config'
import { connectAndStoreSocket } from '../services/gameState'

const router = useRouter()
const interactionError = ref('')

const createForm = ref({
    maxPlayers: 8,
    roundsCount: 3,
    drawingDuration: 60,
    wordsCount: 3,
    choosingWordDuration: 15,
    private: false
})

const handleCreateGame = async () => {
    interactionError.value = ''
    const params = new URLSearchParams({
        private: createForm.value.private.toString(),
        maxPlayers: createForm.value.maxPlayers.toString(),
        roundsCount: createForm.value.roundsCount.toString(),
        wordsCount: createForm.value.wordsCount.toString(),
        choosingWordDuration: createForm.value.choosingWordDuration.toString(),
        drawingDuration: createForm.value.drawingDuration.toString()
    })

    try {
        const wsUrl = `${wsEndpoints.createGame}?${params.toString()}`
        await connectAndStoreSocket(wsUrl)
        router.push({ name: 'game', query: Object.fromEntries(params) })
    } catch (err) {
        console.error('Failed to create game:', err)
        interactionError.value = 'Failed to create game. Please try again.'
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
                class="bg-[#171A21] border border-[#242833] p-8 rounded-2xl shadow-xl hover:border-[#4C8DFF]/50 transition-all duration-300 relative overflow-hidden group">
                <div
                    class="absolute inset-0 bg-gradient-to-br from-[#4C8DFF]/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity">
                </div>

                <h2 class="text-2xl font-bold mb-6 flex items-center gap-2 relative z-10 text-[#E6E6E6]">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-7 h-7 text-[#4C8DFF]" viewBox="0 0 24 24"
                        fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
                        stroke-linejoin="round">
                        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                        <line x1="12" y1="8" x2="12" y2="16"></line>
                        <line x1="8" y1="12" x2="16" y2="12"></line>
                    </svg>
                    Create Game
                </h2>

                <div v-if="interactionError"
                    class="mb-4 p-3 bg-red-500/10 border border-red-500/50 rounded-lg text-red-500 text-sm font-medium relative z-10">
                    {{ interactionError }}
                </div>

                <div class="space-y-5 relative z-10">
                    <div>
                        <label class="block text-xs font-semibold text-[#A0A4AB] uppercase mb-1">Max Players</label>
                        <input type="number" v-model.number="createForm.maxPlayers" min="2" max="20"
                            class="w-full bg-[#0F1115] border border-[#242833] rounded px-3 py-3 text-[#E6E6E6] focus:border-[#4C8DFF] outline-none transition-colors" />
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="block text-xs font-semibold text-[#A0A4AB] uppercase mb-1">Rounds</label>
                            <input type="number" v-model.number="createForm.roundsCount" min="1" max="10"
                                class="w-full bg-[#0F1115] border border-[#242833] rounded px-3 py-3 text-[#E6E6E6] focus:border-[#4C8DFF] outline-none transition-colors" />
                        </div>
                        <div>
                            <label class="block text-xs font-semibold text-[#A0A4AB] uppercase mb-1">Draw Time
                                (s)</label>
                            <input type="number" v-model.number="createForm.drawingDuration" min="30" max="300"
                                class="w-full bg-[#0F1115] border border-[#242833] rounded px-3 py-3 text-[#E6E6E6] focus:border-[#4C8DFF] outline-none transition-colors" />
                        </div>
                    </div>

                    <div>
                        <label class="flex items-center gap-3 cursor-pointer group/check">
                            <div class="relative">
                                <input type="checkbox" v-model="createForm.private" class="sr-only peer" />
                                <div
                                    class="w-10 h-6 bg-[#242833] peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-[#4C8DFF]/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#4C8DFF]">
                                </div>
                            </div>
                            <span
                                class="text-sm font-medium text-[#A0A4AB] group-hover/check:text-[#E6E6E6] transition-colors">Private
                                Room</span>
                        </label>
                    </div>

                    <button @click="handleCreateGame"
                        class="w-full mt-4 py-3.5 bg-[#4C8DFF] hover:bg-[#3b7cdb] text-white font-bold rounded-lg shadow-lg hover:shadow-[#4C8DFF]/20 transition-all active:scale-[0.98]">
                        Create Room
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>
