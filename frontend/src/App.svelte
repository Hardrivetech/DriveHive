<script>
  import { onMount } from 'svelte';

  /** @type {WebSocket | null} */
  let socket = null;
  /** @type {any[]} */
  let messages = $state([]);
  let newMessage = $state("");
  let status = $state("Connecting...");
  let chatWindow = $state();
  let username = $state("");
  let password = $state("");
  let isRegistering = $state(false);
  let isReady = $state(false);
  
  let rooms = ["general", "gaming", "dev"];
  let activeRoom = $state("general");

  // Auto-scroll to bottom when messages change
  $effect(() => {
    if (messages.length && chatWindow) {
      chatWindow.scrollTo({ top: chatWindow.scrollHeight, behavior: 'smooth' });
    }
  });

  onMount(() => {
    // Connect to the Go backend
    socket = new WebSocket("ws://localhost:8080/ws");

    socket.onopen = () => {
      status = "Connected";
      if (isReady) joinRoom(activeRoom);
    };

    socket.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (msg.room_id === activeRoom) {
        messages = [...messages, msg];
      }
    };

    socket.onclose = () => {
      status = "Disconnected";
    };
  });

  /** @param {string} room */
  function joinRoom(room) {
    activeRoom = room;
    messages = [];
    const payload = { type: "join", room_id: room, sender: socket?.url || "init", content: "" };
    socket?.send(JSON.stringify(payload));
  }

  async function handleAuth() {
    const endpoint = isRegistering ? "/register" : "/login";
    const response = await fetch(`http://localhost:8080${endpoint}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });

    if (response.ok) {
      isReady = true;
      if (socket?.readyState === WebSocket.OPEN) joinRoom(activeRoom);
    } else {
      alert(await response.text());
    }
  }

  function sendMessage() {
    if (newMessage.trim() === "" || !socket) return;

    const payload = {
      type: "chat",
      room_id: activeRoom,
      sender: username,
      content: newMessage,
      timestamp: new Date()
    };

    socket.send(JSON.stringify(payload));
    newMessage = "";
  }

  /** @param {SubmitEvent} event */
  async function handleLogin(event) {
    console.log("Attempting login for:", username);
    await handleAuth();
  }

</script>

{#if !isReady}
  <div class="login-screen">
    <div class="login-box">
      <h1>Welcome to DriveHive</h1>
      <p>Enter a username to join the swarm</p>
      <form onsubmit={(e) => { e.preventDefault(); handleLogin(e); }}>
        <input bind:value={username} placeholder="Username..." maxlength="20" />
        <input type="password" bind:value={password} placeholder="Password..." />
        <button type="submit">Join Hive</button>
      </form>
    </div>
  </div>
{:else}
  <div class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-header">Hives</div>
      <nav>
        {#each rooms as room}
          <button 
            class="room-btn" 
            class:active={activeRoom === room}
            onclick={() => joinRoom(room)}
          >
            # {room}
          </button>
        {/each}
      </nav>
      <div class="user-info">
        <div class="status-dot"></div>
        <span>{username}</span>
      </div>
    </aside>

    <main class="container">
      <header>
        <h1># {activeRoom}</h1>
        <span class="status">{status}</span>
      </header>

      <div class="chat-window" bind:this={chatWindow}>
        {#each messages as msg}
          <div class="message">
            <div class="msg-header">
              <span class="sender">{msg.sender}</span>
              <span class="timestamp">{new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
            </div>
            <span class="content">{msg.content}</span>
          </div>
        {/each}
      </div>

      <form 
        onsubmit={(e) => { e.preventDefault(); sendMessage(); }} 
        class="input-area"
      >
        <input bind:value={newMessage} placeholder="Message #{activeRoom}" />
      </form>
    </main>
  </div>
{/if}

<style>
  :global(body) { background: #36393f; color: #dcddde; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; overflow: hidden; }
  
  .login-screen { display: flex; align-items: center; justify-content: center; height: 100vh; background: #2f3136; }
  .login-box { background: #36393f; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.2); text-align: center; }
  
  .app-layout { display: flex; height: 100vh; }
  
  .sidebar { width: 240px; background: #2f3136; display: flex; flex-direction: column; }
  .sidebar-header { padding: 15px; font-weight: bold; border-bottom: 1px solid #26272b; color: #fff; }
  .room-btn { background: none; border: none; color: #8e9297; text-align: left; padding: 10px 15px; cursor: pointer; font-size: 1rem; }
  .room-btn:hover { background: #393c43; color: #dcddde; }
  .room-btn.active { background: #393c43; color: #fff; }
  
  .user-info { margin-top: auto; padding: 15px; background: #292b2f; display: flex; align-items: center; gap: 10px; }
  .status-dot { width: 10px; height: 10px; background: #43b581; border-radius: 50%; }

  .container { flex: 1; display: flex; flex-direction: column; background: #36393f; }
  header { padding: 10px 20px; border-bottom: 1px solid #26272b; display: flex; justify-content: space-between; align-items: center; }
  header h1 { font-size: 1.2rem; margin: 0; color: #fff; }
  
  .chat-window { flex: 1; overflow-y: auto; padding: 20px; display: flex; flex-direction: column; gap: 15px; }
  .message { display: flex; flex-direction: column; }
  .msg-header { display: flex; align-items: baseline; gap: 8px; }
  .sender { color: #fff; font-weight: 600; }
  .timestamp { color: #72767d; font-size: 0.75rem; }
  .content { color: #dcddde; line-height: 1.4; }

  .input-area { display: flex; gap: 10px; }
  input { flex: 1; padding: 12px; border-radius: 8px; border: none; background: #40444b; color: #dcddde; margin: 20px; outline: none; }
  
  button[type="submit"] { background: #5865f2; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; margin-top: 10px; }
</style>