export interface PublicGame {
    id: string
    private: boolean
    playersCount: number
    maxPlayers: number
    started: boolean
}

import { endpoints } from '../config'

export const api = {
    getPublicGames: async (): Promise<PublicGame[]> => {
        try {
            const response = await fetch(endpoints.publicGames, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            if (!response.ok) {
                throw new Error(`Failed to fetch games: ${response.statusText}`)
            }
            return await response.json()
        } catch (error) {
            console.error('Error fetching public games:', error)
            throw error
        }
    }
}
