<script>
  import { onMount } from 'svelte';
  import { connectWS } from '$lib/api/websocket.js';
  import { fetchSessions } from '$lib/api/sessions.js';
  import { getRPCStatus } from '$lib/api/rpc.js';
  import { activeSession, sessions } from '$lib/stores/session.svelte.js';
  import { userScrolledUp, newMessageCount } from '$lib/stores/messages.svelte.js';
  import { setRpcRunning } from '$lib/stores/rpc.svelte.js';
  import { sidebarOpen, newSessionModalOpen } from '$lib/stores/ui.svelte.js';
  import { ws } from '$lib/stores/ws.svelte.js';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import HeaderBar from '$lib/components/HeaderBar.svelte';
  import ChatArea from '$lib/components/ChatArea.svelte';
  import NewSessionModal from '$lib/components/NewSessionModal.svelte';

  let isMobile = $state(false);

  onMount(() => {
    // Check if mobile
    isMobile = window.innerWidth <= 768;

    // Listen for resize
    const handleResize = () => {
      isMobile = window.innerWidth <= 768;
    };
    window.addEventListener('resize', handleResize);

    // Connect WebSocket
    connectWS();

    // Fetch initial session list
    fetchSessions()
      .then(list => sessions.set(list))
      .catch(e => console.error('Failed to fetch sessions:', e));

    // Sync RPC status from server (restores state after page reload)
    getRPCStatus()
      .then(data => {
        if (data.sessions) {
          for (const [sessionId, running] of Object.entries(data.sessions)) {
            if (running) {
              setRpcRunning(sessionId, true);
            }
          }
        }
      })
      .catch(() => {});

    // Re-subscribe to active session on reload (scrolls to bottom)
    let savedSession = null;
    activeSession.subscribe(id => { savedSession = id; })();
    if (savedSession) {
      const trySubscribe = () => {
        let socket = null;
        ws.subscribe(s => { socket = s; })();
        if (socket && socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'subscribe', session_id: savedSession }));
          userScrolledUp.set(false);
          newMessageCount.set(0);
        } else {
          // WS not ready yet, retry after a short delay
          setTimeout(trySubscribe, 200);
        }
      };
      trySubscribe();
    }

    // Refresh sessions periodically
    const interval = setInterval(() => {
      fetchSessions()
        .then(list => sessions.set(list))
        .catch(() => {});
    }, 5000);

    return () => {
      clearInterval(interval);
      window.removeEventListener('resize', handleResize);
    };
  });

  function showNewSessionModal() {
    newSessionModalOpen.set(true);
  }
</script>

<div class="flex h-screen">
  <!-- Sidebar overlay (mobile) -->
  {#if isMobile}
    <div
      class="sidebar-overlay"
      class:hidden={!$sidebarOpen}
      onclick={() => sidebarOpen.set(false)}
    ></div>
  {/if}

  <!-- Sidebar -->
  <div
    class="sidebar h-full"
    class:hidden={isMobile && !$sidebarOpen}
  >
    <Sidebar onNewSession={showNewSessionModal} />
  </div>

  <!-- Main -->
  <div class="flex-1 flex flex-col main-content">
    <HeaderBar />
    <ChatArea />
  </div>

  <!-- New Session Modal -->
  <NewSessionModal />
</div>

<style>
  @media (min-width: 769px) {
    .sidebar {
      position: relative !important;
      left: 0 !important;
    }
    .sidebar-overlay {
      display: none !important;
    }
  }
</style>
