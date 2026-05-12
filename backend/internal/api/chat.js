import { writable, get } from 'svelte/store';

/** @type {import('svelte/store').Writable<WebSocket | null>} */
export const socket = writable(null);
/** @type {import('svelte/store').Writable<any[]>} */
export const messages = writable([]);
/** @type {import('svelte/store').Writable<string | null>} */
export const currentChannelId = writable(null);
/** @type {import('svelte/store').Writable<boolean>} */
export const isConnected = writable(false);

/**
 * Initialize the WebSocket connection using the user's JWT
 */
export function connect(token) {
    const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

    ws.onopen = () => {
        isConnected.set(true);
        console.log("Connected to DriveHive Hub");
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        // The backend sends both history (during join) and live messages here
        messages.update(prev => [...prev, data]);
    };

    ws.onclose = () => {
        isConnected.set(false);
        socket.set(null);
        console.warn("WebSocket disconnected");
    };

    socket.set(ws);
}

/**
 * Tells the backend to move the user into a specific channel's room
 */
export function switchChannel(channelId) {
    const ws = get(socket);
    
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.error("Cannot join channel: WebSocket is not connected");
        return;
    }

    currentChannelId.set(channelId);
    messages.set([]); // Clear UI for the new channel

    ws.send(JSON.stringify({
        type: "join",
        room_id: channelId
    }));
}