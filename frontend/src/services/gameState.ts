import { ref } from 'vue'

export const activeSocket = ref<WebSocket | null>(null)
export const bufferedMessages = ref<MessageEvent[]>([])

export const connectAndStoreSocket = (url: string): Promise<void> => {
    return new Promise((resolve, reject) => {
        const ws = new WebSocket(url)
        ws.binaryType = 'arraybuffer'
        const tempBuffer: MessageEvent[] = []

        // Buffer messages until the view takes over
        const msgListener = (e: MessageEvent) => {
            console.log('Buffering message:', e.data)
            tempBuffer.push(e)
        }
        ws.addEventListener('message', msgListener)

        ws.onopen = () => {
            console.log('Socket opened in service for:', url)
            activeSocket.value = ws
            bufferedMessages.value = tempBuffer
            resolve()
        }

        ws.onclose = (event) => {
            console.log('Socket closed in service:', event.code, event.reason)
            // If we fail to connect initially (e.g. 400), we reject
            reject(event)
        }

        ws.onerror = (err) => {
            console.error('Socket error in service:', err)
            // onerror usually precedes onclose
        }
    })
}

export const clearGameState = () => {
    activeSocket.value = null
    bufferedMessages.value = []
}
