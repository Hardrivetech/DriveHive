<script>
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  let username = '';
  let password = '';
  let confirmPassword = '';
  let error = '';
  let loading = false;

  async function handleRegister() {
    error = '';
    
    if (password !== confirmPassword) {
      error = "Passwords do not match";
      return;
    }

    loading = true;
    try {
      const response = await fetch('http://localhost:8080/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
      });

      if (response.ok) {
        // Notify parent to switch back to login or show success
        dispatch('success');
      } else {
        const errMsg = await response.text();
        error = errMsg || 'Registration failed';
      }
    } catch (err) {
      error = "Cannot connect to the backend server.";
      console.error(err);
    } finally {
      loading = false;
    }
  }
</script>

<div class="register-container">
  <h2>Join DriveHive</h2>
  <form on:submit|preventDefault={handleRegister}>
    <div class="input-group">
      <label for="reg-username">Username</label>
      <input id="reg-username" type="text" bind:value={username} required disabled={loading} />
    </div>
    
    <div class="input-group">
      <label for="reg-password">Password</label>
      <input id="reg-password" type="password" bind:value={password} required disabled={loading} />
    </div>

    <div class="input-group">
      <label for="confirm-password">Confirm Password</label>
      <input id="confirm-password" type="password" bind:value={confirmPassword} required disabled={loading} />
    </div>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <button type="submit" disabled={loading}>
      {loading ? 'Creating Account...' : 'Register'}
    </button>
  </form>
  
  <p class="toggle-text">
    Already have an account? 
    <button class="link-btn" on:click={() => dispatch('toggle')}>
      Login here
    </button>
  </p>
</div>