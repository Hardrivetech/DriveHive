<script>
  import { onMount } from 'svelte';
  import Register from './lib/Register.svelte';
  import Sidebar from './lib/Sidebar.svelte';
  import { connect, messages, socket, currentChannelId, switchChannel, isConnected } from './chat.js';

  let newMessage = $state("");
  let status = $derived($isConnected ? "Connected" : "Connecting...");
  let chatWindow = $state();
  let username = $state("");
  let password = $state("");
  let isRegistering = $state(false);
  let isReady = $state(false);
  
  let rooms = [
    { id: "general", name: "general" },
    { id: "gaming", name: "gaming" },
    { id: "dev", name: "dev" }
  ];

  // Auto-scroll to bottom when messages change
  $effect(() => {
    if ($messages.length && chatWindow) {
      chatWindow.scrollTo({ top: chatWindow.scrollHeight, behavior: 'smooth' });
    }
  });

  // Join default room when socket is ready
  $effect(() => {
    if ($isConnected && isReady && !$currentChannelId) {
      switchChannel('general');
    }
  });

  onMount(async () => {
    const savedUser = localStorage.getItem("dh_user");
    if (savedUser) {
      const data = JSON.parse(savedUser);
      // Validate token with backend
      const res = await fetch("http://localhost:8080/validate",{
        headers: {"Authorization": `Bearer ${data.token}`}
      });

      if (res.ok) {
        username = data.username;
        isReady = true;
        connect(data.token);
      } else {
        localStorage.removeItem("dh_user");
      }
    }
  });

  async function handleAuth() {
    const endpoint = isRegistering ? "/register" : "/login";
    const response = await fetch(`http://localhost:8080${endpoint}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });

    if (response.ok) {
      if (!isRegistering) {
        const data = await response.json();
        localStorage.setItem("dh_user", JSON.stringify(data));
        connect(data.token);
        isReady = true;
      } else {
        isRegistering = false;
        alert("Registration successful! Please login with your new account.");
      }
    } else {
      alert(await response.text());
    }
  }

  function sendMessage() {
    if (newMessage.trim() === "" || !$socket) return;

    const payload = {
      type: "chat",
      room_id: $currentChannelId,
      sender: username,
      content: newMessage,
      timestamp: new Date()
    };

    $socket.send(JSON.stringify(payload));
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
      {#if isRegistering}
        <Register 
          on:success={() => isRegistering = false} 
          on:toggle={() => isRegistering = false} 
        />
      {:else}
        <h1>Welcome to DriveHive</h1>
        <p>Enter your credentials to join the swarm</p>
        <form onsubmit={(e) => { e.preventDefault(); handleLogin(e); }}>
          <input bind:value={username} placeholder="Username..." maxlength="20" />
          <input type="password" bind:value={password} placeholder="Password..." />
          <button type="submit">Join Hive</button>
        </form>
        <p class="toggle-text">
          New here? <button class="link-btn" onclick={() => isRegistering = true}>Create an account</button>
        </p>
      {/if}
    </div>
  </div>
{:else}
  <div class="app-layout">
    <Sidebar channels={rooms} {username} />

    <main class="container">
      <header>
        <h1># {$currentChannelId || 'Select a channel'}</h1>
        <span class="status">{status}</span>
      </header>

      <div class="chat-window" bind:this={chatWindow}>
        {#each $messages as msg}
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
        <input bind:value={newMessage} placeholder="Message #{$currentChannelId || ''}" />
      </form>
    </main>
  </div>
{/if}

<style>
  :global(body) { background: #36393f; color: #dcddde; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; overflow: hidden; }
  
  .login-screen { display: flex; align-items: center; justify-content: center; height: 100vh; background: #2f3136; }
  .login-box { background: #36393f; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.2); text-align: center; }
  
  .app-layout { display: flex; height: 100vh; }

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
  
  .toggle-text { margin-top: 20px; font-size: 0.9rem; color: #b9bbbe; }
  .link-btn { background: none; border: none; color: #00aff4; cursor: pointer; padding: 0; font: inherit; }
  .link-btn:hover { text-decoration: underline; }
</style>