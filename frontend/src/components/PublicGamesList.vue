<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { api, type PublicGame } from '../services/api'
import { wsEndpoints } from '../config'
import { connectAndStoreSocket } from '../services/gameState'

const router = useRouter()
const games = ref<PublicGame[]>([])
const loading = ref(false)
const error = ref('')
const joiningId = ref('')

const fetchGames = async () => {
    loading.value = true
    error.value = ''
    try {
        games.value = await api.getPublicGames()
    } catch (err) {
        error.value = 'Failed to load games'
    } finally {
        loading.value = false
    }
}

const handleJoin = async (gameId: string) => {
    joiningId.value = gameId
    try {
        const wsUrl = wsEndpoints.joinRoom(gameId)
        await connectAndStoreSocket(wsUrl)
        router.push({ name: 'game' })
    } catch (err) {
        console.error('Failed to join game:', err)
        error.value = 'Failed to join game. It might be full or finished.'
    } finally {
        joiningId.value = ''
    }
}

let intervalId: number
onMounted(() => {
    fetchGames()
    intervalId = setInterval(fetchGames, 5000) // Auto-refresh every 5s
})

onUnmounted(() => {
    if (intervalId) clearInterval(intervalId)
})
</script>

<template>
    <div
        class="bg-[#171A21] border border-[#242833] rounded-2xl overflow-hidden shadow-xl flex flex-col h-full max-h-[600px]">
        <div
            class="p-6 border-b border-[#242833] flex justify-between items-center bg-[#171A21]/50 backdrop-blur-sm sticky top-0 z-10">
            <h2 class="text-xl font-bold flex items-center gap-3">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6 text-[#f39c12]" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <polygon points="10 8 16 12 10 16 10 8"></polygon>
                </svg>
                Public Games
                <span v-if="games.length > 0" class="text-xs bg-[#242833] text-[#A0A4AB] px-2 py-0.5 rounded-full">{{
                    games.length }}</span>
            </h2>
            <button @click="fetchGames" :disabled="loading"
                class="text-[#A0A4AB] hover:text-[#4C8DFF] transition-colors p-2 rounded-lg hover:bg-[#242833]">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" :class="{ 'animate-spin': loading }"
                    viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
                    stroke-linejoin="round">
                    <path d="M23 4v6h-6"></path>
                    <path d="M1 20v-6h6"></path>
                    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
                </svg>
            </button>
        </div>

        <div class="overflow-y-auto custom-scrollbar grow relative">
            <div v-if="loading && games.length === 0"
                class="flex flex-col items-center justify-center py-12 text-[#A0A4AB]">
                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-[#4C8DFF] mb-4"></div>
                <p>Loading games...</p>
            </div>

            <div v-else-if="error" class="p-8 text-center bg-red-500/5 m-4 rounded-xl border border-red-500/20">
                <p class="text-red-400 mb-4">{{ error }}</p>
                <button @click="fetchGames"
                    class="px-4 py-2 bg-[#242833] hover:bg-[#2c313c] rounded-lg text-sm transition-colors">
                    Try Again
                </button>
            </div>

            <div v-else-if="games.length === 0"
                class="flex flex-col items-center justify-center py-16 text-center px-6">
                <div class="w-16 h-16 bg-[#242833] rounded-full flex items-center justify-center mb-4 text-[#A0A4AB]">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8" viewBox="0 0 24 24" fill="none"
                        stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <rect x="2" y="13" width="8" height="8" rx="2" ry="2"></rect>
                        <rect x="14" y="13" width="8" height="8" rx="2" ry="2"></rect>
                        <rect x="8" y="3" width="8" height="8" rx="2" ry="2"></rect>
                    </svg>
                </div>
                <h3 class="text-lg font-bold text-[#E6E6E6] mb-2">No Games Found</h3>
                <p class="text-[#A0A4AB] text-sm max-w-xs">There are no public games currently active. Be the first to
                    start one!</p>
            </div>

            <table v-else class="w-full text-left border-collapse">
                <thead class="bg-[#171A21] sticky top-0 z-0 text-xs uppercase text-[#A0A4AB] font-semibold">
                    <tr>
                        <th class="p-4 pl-6">Room ID</th>
                        <th class="p-4 text-center">Players</th>
                        <th class="p-4 text-center">Status</th>
                        <th class="p-4 pr-6 text-right">Action</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-[#242833]">
                    <tr v-for="game in games" :key="game.id" class="hover:bg-[#1a1d26] transition-colors group">
                        <td class="p-4 pl-6 font-mono text-[#f39c12]">{{ game.id }}</td>
                        <td class="p-4 text-center text-[#E6E6E6]">
                            <span
                                :class="{ 'text-[#2ecc71]': game.playersCount < game.maxPlayers, 'text-red-400': game.playersCount >= game.maxPlayers }">
                                {{ game.playersCount }}
                            </span>
                            <span class="text-[#A0A4AB]">/{{ game.maxPlayers }}</span>
                        </td>
                        <td class="p-4 text-center">
                            <span v-if="game.started"
                                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
                                Playing
                            </span>
                            <span v-else
                                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-[#2ecc71]/10 text-[#2ecc71] border border-[#2ecc71]/20">
                                Waiting
                            </span>
                        </td>
                        <td class="p-4 pr-6 text-right">
                            <button @click="handleJoin(game.id)"
                                :disabled="joiningId === game.id || game.playersCount >= game.maxPlayers"
                                class="px-4 py-2 bg-[#2ecc71] hover:bg-[#27ae60] text-white text-sm font-bold rounded-lg transition-all shadow hover:shadow-[#2ecc71]/20 disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none min-w-[80px]">
                                <span v-if="joiningId === game.id" class="flex items-center justify-center gap-2">
                                    <div
                                        class="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin">
                                    </div>
                                </span>
                                <span v-else>Join</span>
                            </button>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
    width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
    background: #171A21;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
    background: #242833;
    border-radius: 3px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: #323846;
}
</style>
