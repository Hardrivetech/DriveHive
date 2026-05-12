<script>
  import { switchChannel, currentChannelId } from '../stores/chat.js';
  
  // This array would typically be populated by a fetch call to /channels?hive_id=...
  /** @type {Array<{id: string, name: string}>} */
  export let channels = [];
  /** @type {string} */
  export let username = "";

  /**
   * Handles the UI click event to change the active room
   * @param {string} id
   */
  function handleChannelClick(id) {
    switchChannel(id);
  }
</script>

<nav class="sidebar">
  <header class="sidebar-header">
    <h2>Channels</h2>
  </header>

  <ul class="channel-list">
    {#each channels as channel}
      <li class="channel-item">
        <button 
          class="channel-link" 
          class:active={$currentChannelId === channel.id}
          on:click={() => handleChannelClick(channel.id)}
        >
          <span class="prefix">#</span>
          <span class="name">{channel.name}</span>
        </button>
      </li>
    {/each}
  </ul>

  <footer class="user-info">
    <div class="status-dot"></div>
    <span class="username-text">{username}</span>
  </footer>
</nav>

<style>
  .sidebar {
    width: 240px;
    background-color: #2f3136;
    display: flex;
    flex-direction: column;
    height: 100vh;
    border-right: 1px solid #202225;
  }

  .sidebar-header {
    padding: 12px 16px;
    box-shadow: 0 1px 0 rgba(0,0,0,0.2);
    color: white;
  }

  .channel-list {
    list-style: none;
    padding: 8px;
    margin: 0;
  }

  .channel-link {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 6px 8px;
    margin-bottom: 2px;
    background: none;
    border: none;
    border-radius: 4px;
    color: #8e9297;
    cursor: pointer;
    text-align: left;
    font-size: 1rem;
  }

  .channel-link:hover {
    background-color: #393c43;
    color: #dcddde;
  }

  .channel-link.active {
    background-color: #40444b;
    color: #fff;
  }

  .prefix {
    font-size: 1.2rem;
    margin-right: 6px;
    color: #72767d;
  }

  .user-info {
    margin-top: auto;
    padding: 12px 16px;
    background-color: #292b2f;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-dot {
    width: 10px;
    height: 10px;
    background-color: #43b581;
    border-radius: 50%;
  }

  .username-text {
    color: #fff;
    font-weight: 500;
    font-size: 0.9rem;
  }
</style>