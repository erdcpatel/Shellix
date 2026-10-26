/**
 * Shellix Web Workbench — Modular Client Engine
 * High-performance, zero-dependency client with in-memory caching and reactive UI
 */

(function () {
  'use strict';

  // =========================================================================
  // Application State Store
  // =========================================================================
  const state = {
    activeTab: 'review',
    diffMode: 'split', // 'split' | 'unified'
    ignoreWhitespace: false,
    workspace: {
      name: 'workspace',
      path: '',
      branch: 'main',
      ahead: 0,
      version: '0.1.0-alpha',
    },
    git: {
      files: [],
      activeFile: null,
      activeDiff: null,
    },
    flows: [],
    activeFlow: null,
    search: {
      query: '',
      ext: '',
      results: null,
      isSearching: false,
    },
    httpdiff: {
      results: null,
      activeView: 'badges',
    },
    dataview: {
      summary: null,
      selectedColumns: [],
      filter: '',
      isUploaded: false,
    },
  };

  // =========================================================================
  // API Client Helpers
  // =========================================================================
  function getCSRFToken() {
    const meta = document.querySelector('meta[name="shellix-csrf-token"]');
    return meta ? meta.getAttribute('content') : '';
  }

  async function fetchJSON(url, options = {}) {
    try {
      const token = getCSRFToken();
      const headers = {
        ...(options.headers || {}),
      };
      if (token) {
        headers['X-CSRF-Token'] = token;
        headers['X-Shellix-Session'] = token;
      }
      const res = await fetch(url, { ...options, headers });
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`);
      }
      return await res.json();
    } catch (err) {
      console.error(`API Error on ${url}:`, err);
      showToast(`API Error: ${err.message}`, 'error');
      return null;
    }
  }

  // =========================================================================
  // Initialization & Boot
  // =========================================================================
  document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initNavigation();
    initShortcuts();
    initReviewView();
    initFlowView();
    initSearchView();
    initVaultView();
    initDoctorView();
    initHTTPDiffView();
    initDataView();
    initResourceMeter();
    initCommandPalette();

    // Initial data hydration
    loadSystemInfo();
    loadGitStatus();
    loadFlows();

    // Check initial hash route
    const hash = window.location.hash.replace('#', '');
    if (hash && ['review', 'flow', 'search', 'vault', 'doctor', 'httpdiff', 'dataview'].includes(hash)) {
      switchTab(hash);
    }
  });

  // =========================================================================
  // Theme Engine (Light / Dark)
  // =========================================================================
  function initTheme() {
    const urlParams = new URLSearchParams(window.location.search);
    const queryTheme = urlParams.get('theme');
    const savedTheme = localStorage.getItem('shellix_theme');
    let theme = queryTheme || savedTheme;
    if (!theme) {
      theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
    }
    applyTheme(theme);

    document.getElementById('themeToggleBtn')?.addEventListener('click', () => {
      const current = document.body.classList.contains('light-theme') ? 'light' : 'dark';
      const next = current === 'dark' ? 'light' : 'dark';
      applyTheme(next);
      showToast(`Switched to ${next} theme`, 'info');
    });
  }

  function applyTheme(theme) {
    const sunIcon = document.getElementById('themeIconSun');
    const moonIcon = document.getElementById('themeIconMoon');

    if (theme === 'light') {
      document.body.classList.remove('dark-theme');
      document.body.classList.add('light-theme');
      if (sunIcon) sunIcon.style.display = 'none';
      if (moonIcon) moonIcon.style.display = 'block';
    } else {
      document.body.classList.remove('light-theme');
      document.body.classList.add('dark-theme');
      if (sunIcon) sunIcon.style.display = 'block';
      if (moonIcon) moonIcon.style.display = 'none';
    }
    localStorage.setItem('shellix_theme', theme);
  }

  // =========================================================================
  // Navigation & Tab Routing
  // =========================================================================
  function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach((btn) => {
      btn.addEventListener('click', () => {
        const tab = btn.getAttribute('data-tab');
        switchTab(tab);
      });
    });

    // Sidebar collapse toggler
    const sidebarToggleBtn = document.getElementById('sidebarToggleBtn');
    const appContainer = document.querySelector('.app-container');

    if (localStorage.getItem('shellix_sidebar_collapsed') === 'true') {
      appContainer?.classList.add('sidebar-collapsed');
    }

    sidebarToggleBtn?.addEventListener('click', () => {
      appContainer?.classList.toggle('sidebar-collapsed');
      const isCollapsed = appContainer?.classList.contains('sidebar-collapsed');
      localStorage.setItem('shellix_sidebar_collapsed', isCollapsed);
      showToast(isCollapsed ? 'Sidebar collapsed (press [ to restore)' : 'Sidebar expanded', 'info');
      setTimeout(() => {
        state.dataview?.tabulatorInstance?.redraw(true);
      }, 250);
    });

    document.getElementById('refreshBtn')?.addEventListener('click', () => {
      showToast('Refreshing workspace state...', 'info');
      loadSystemInfo();
      loadGitStatus();
    });
  }

  function switchTab(tabId) {
    state.activeTab = tabId;

    // Update nav items
    document.querySelectorAll('.nav-item').forEach((item) => {
      if (item.getAttribute('data-tab') === tabId) {
        item.classList.add('active');
      } else {
        item.classList.remove('active');
      }
    });

    // Update tab panes
    document.querySelectorAll('.tab-pane').forEach((pane) => {
      if (pane.id === `pane-${tabId}`) {
        pane.classList.add('active');
      } else {
        pane.classList.remove('active');
      }
    });

    // Trigger tab specific refresh
    if (tabId === 'vault') {
      loadVaultData();
    } else if (tabId === 'doctor') {
      loadDoctorData();
    } else if (tabId === 'httpdiff') {
      if (!state.httpdiff.results) runHTTPDiff();
    } else if (tabId === 'dataview') {
      if (!state.dataview.summary) loadDataView();
    }
  }

  // =========================================================================
  // Module 1: Pre-Flight Review & Side-by-Side Diffs
  // =========================================================================
  function initReviewView() {
    const splitBtn = document.getElementById('diffModeSplitBtn');
    const unifiedBtn = document.getElementById('diffModeUnifiedBtn');
    const wsCheck = document.getElementById('ignoreWhitespaceCheck');
    const fileFilter = document.getElementById('fileFilterInput');
    const copyBtn = document.getElementById('copyDiffBtn');

    splitBtn?.addEventListener('click', () => {
      state.diffMode = 'split';
      splitBtn.classList.add('active');
      unifiedBtn.classList.remove('active');
      renderActiveDiff();
    });

    unifiedBtn?.addEventListener('click', () => {
      state.diffMode = 'unified';
      unifiedBtn.classList.add('active');
      splitBtn.classList.remove('active');
      renderActiveDiff();
    });

    wsCheck?.addEventListener('change', (e) => {
      state.ignoreWhitespace = e.target.checked;
      if (state.git.activeFile) {
        loadDiffForFile(state.git.activeFile);
      }
    });

    fileFilter?.addEventListener('input', (e) => {
      renderFileList(e.target.value.trim().toLowerCase());
    });

    copyBtn?.addEventListener('click', () => {
      if (state.git.activeDiff) {
        navigator.clipboard.writeText(state.git.activeDiff.raw);
        showToast('Diff copied to clipboard!', 'success');
      }
    });
  }

  async function loadGitStatus() {
    const data = await fetchJSON('/api/git/status');
    if (!data) return;

    state.git.files = data.files || [];
    const countEl = document.getElementById('changedFilesCount');
    const badgeEl = document.getElementById('reviewBadge');

    if (countEl) countEl.textContent = state.git.files.length;
    if (badgeEl) badgeEl.textContent = state.git.files.length;

    renderFileList();

    // Auto-select first file if none selected
    if (state.git.files.length > 0 && !state.git.activeFile) {
      selectFileForDiff(state.git.files[0]);
    }
  }

  function renderFileList(filterQuery = '') {
    const container = document.getElementById('fileListContainer');
    if (!container) return;

    container.innerHTML = '';

    const filtered = state.git.files.filter((f) =>
      f.path.toLowerCase().includes(filterQuery)
    );

    if (filtered.length === 0) {
      container.innerHTML = `<div class="empty-state-sm">No changed files found.</div>`;
      return;
    }

    filtered.forEach((file) => {
      const item = document.createElement('div');
      item.className = 'file-item';
      if (state.git.activeFile && state.git.activeFile.path === file.path) {
        item.classList.add('active');
      }

      item.innerHTML = `
        <span class="file-name" title="${file.path}">${formatFilePath(file.path)}</span>
        <span class="file-status-badge status-${file.status}">${file.status}</span>
      `;

      item.addEventListener('click', () => selectFileForDiff(file));
      container.appendChild(item);
    });
  }

  function formatFilePath(path) {
    const parts = path.split('/');
    if (parts.length > 2) {
      return `.../${parts.slice(-2).join('/')}`;
    }
    return path;
  }

  async function selectFileForDiff(file) {
    state.git.activeFile = file;

    // Update active highlight in file list
    document.querySelectorAll('.file-item').forEach((el) => {
      if (el.querySelector('.file-name').getAttribute('title') === file.path) {
        el.classList.add('active');
      } else {
        el.classList.remove('active');
      }
    });

    await loadDiffForFile(file);
  }

  async function loadDiffForFile(file) {
    const pathEl = document.getElementById('activeDiffFilePath');
    const statsEl = document.getElementById('activeDiffStats');
    if (pathEl) pathEl.textContent = file.path;

    const query = new URLSearchParams({
      file: file.path,
      staged: file.is_staged ? 'true' : 'false',
      ignore_ws: state.ignoreWhitespace ? 'true' : 'false',
    });

    const diffData = await fetchJSON(`/api/git/diff?${query.toString()}`);
    if (!diffData) return;

    state.git.activeDiff = diffData;
    if (statsEl) {
      statsEl.innerHTML = `
        <span class="diff-stat-add">+${diffData.additions || 0}</span>
        <span class="diff-stat-del">-${diffData.deletions || 0}</span>
      `;
    }

    renderActiveDiff();
  }

  function renderActiveDiff() {
    const container = document.getElementById('diffContentContainer');
    if (!container || !state.git.activeDiff) return;

    const diff = state.git.activeDiff;

    if (!diff.lines || diff.lines.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <h3>No Line Changes</h3>
          <p>File is unmodified or contains non-text binary data.</p>
        </div>`;
      return;
    }

    if (state.diffMode === 'split') {
      renderSplitDiff(container, diff);
    } else {
      renderUnifiedDiff(container, diff);
    }
  }

  function renderSplitDiff(container, diff) {
    let html = `<table class="split-diff-table"><tbody>`;

    diff.split_rows.forEach((row) => {
      if (row.is_hunk) {
        html += `
          <tr>
            <td colspan="4" class="diff-line-hunk">${escapeHTML(row.left_content || row.right_content)}</td>
          </tr>`;
        return;
      }

      const leftCls = row.left_type === 'del' ? 'diff-line-del' : '';
      const rightCls = row.right_type === 'add' ? 'diff-line-add' : '';

      html += `
        <tr>
          <td class="diff-line-num ${leftCls}">${row.left_num || ''}</td>
          <td class="diff-pane-left ${leftCls}">${escapeHTML(row.left_content || '')}</td>
          <td class="diff-line-num ${rightCls}">${row.right_num || ''}</td>
          <td class="diff-pane-right ${rightCls}">${escapeHTML(row.right_content || '')}</td>
        </tr>`;
    });

    html += `</tbody></table>`;
    container.innerHTML = html;
  }

  function renderUnifiedDiff(container, diff) {
    let html = `<table class="split-diff-table"><tbody>`;

    diff.lines.forEach((line) => {
      let cls = '';
      if (line.type === 'add') cls = 'diff-line-add';
      if (line.type === 'del') cls = 'diff-line-del';
      if (line.type === 'hunk') cls = 'diff-line-hunk';

      html += `
        <tr class="${cls}">
          <td class="diff-line-num">${line.old_num || ''}</td>
          <td class="diff-line-num">${line.new_num || ''}</td>
          <td colspan="2">${escapeHTML(line.content)}</td>
        </tr>`;
    });

    html += `</tbody></table>`;
    container.innerHTML = html;
  }

  function downloadBlob(content, filename, contentType) {
    const blob = new Blob([content], { type: contentType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  // =========================================================================
  // Module 2: Automation Flows View
  // =========================================================================
  function initFlowView() {
    const flowSelect = document.getElementById('flowSelect');
    const runBtn = document.getElementById('runSelectedFlowBtn');

    flowSelect?.addEventListener('change', (e) => {
      const name = e.target.value;
      const flow = state.flows.find((f) => f.name === name);
      state.activeFlow = flow;
      renderFlowBlueprint(flow);
    });

    runBtn?.addEventListener('click', async () => {
      if (!state.activeFlow) {
        showToast('Please select a flow first', 'warning');
        return;
      }
      showToast(`Triggered execution: ${state.activeFlow.name}`, 'info');
      const res = await fetchJSON('/api/flow/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ flow: state.activeFlow.name }),
      });
      if (res && res.success) {
        showToast(`Flow '${state.activeFlow.name}' completed in ${res.duration}`, 'success');
      }
    });
  }

  async function loadFlows() {
    const data = await fetchJSON('/api/flows');
    if (!data || !data.flows) return;

    state.flows = data.flows;
    const flowSelect = document.getElementById('flowSelect');
    const badge = document.getElementById('flowsBadge');

    if (badge) badge.textContent = state.flows.length;

    if (flowSelect) {
      flowSelect.innerHTML = `<option value="">Select a workflow (${state.flows.length} available)...</option>`;
      state.flows.forEach((f) => {
        const opt = document.createElement('option');
        opt.value = f.name;
        opt.textContent = `${f.name} (${f.steps_count} steps)`;
        flowSelect.appendChild(opt);
      });
    }
  }

  function renderFlowBlueprint(flow) {
    const container = document.getElementById('flowBlueprintContainer');
    if (!container || !flow) return;

    let html = `
      <div style="margin-bottom: 20px;">
        <h3 style="font-size: 15px; font-weight: 600; color: var(--text-primary);">${flow.name}</h3>
        <p style="color: var(--text-muted); font-size: 12px; margin-top: 2px;">${flow.description || 'No description provided'}</p>
      </div>`;

    if (flow.waves && flow.waves.length > 0) {
      flow.waves.forEach((wave) => {
        const typeStr = wave.is_parallel ? 'Concurrent Parallel Wave' : 'Sequential Step';
        html += `
          <div class="flow-wave-card">
            <div class="wave-title">
              <span>🌊 Wave ${wave.wave_index}</span>
              <span style="font-size: 11px; color: var(--text-muted); font-weight: 400;">(${typeStr})</span>
            </div>
            <div class="wave-steps">`;

        wave.steps.forEach((s) => {
          html += `
            <div class="step-card">
              <div>
                <span class="step-name">${s.name}</span>
                <span class="step-cmd">${escapeHTML(s.run)}</span>
              </div>
              <div style="font-size: 11px; color: var(--text-muted);">
                ${s.timeout ? `timeout: ${s.timeout}` : ''}
              </div>
            </div>`;
        });

        html += `</div></div>`;
      });
    }

    container.innerHTML = html;
  }

  // =========================================================================
  // Module 3: Search Explorer View
  // =========================================================================
  function initSearchView() {
    const searchBtn = document.getElementById('executeSearchBtn');
    const queryInput = document.getElementById('searchQueryInput');
    const extInput = document.getElementById('searchExtInput');

    searchBtn?.addEventListener('click', performSearch);
    queryInput?.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') performSearch();
    });
  }

  async function performSearch() {
    const q = document.getElementById('searchQueryInput')?.value.trim();
    const scope = document.getElementById('searchScopeInput')?.value.trim() || '';
    const ext = document.getElementById('searchExtInput')?.value.trim() || '';
    if (!q) {
      showToast('Please enter a search query', 'warning');
      return;
    }

    const scopeDesc = scope ? `in ${scope}` : 'across workspace';
    showToast(`Searching for "${q}" ${scopeDesc}...`, 'info');
    const url = `/api/search?q=${encodeURIComponent(q)}&scope=${encodeURIComponent(scope)}&ext=${encodeURIComponent(ext)}&by_dir=true`;
    const data = await fetchJSON(url);
    if (!data) return;

    state.search.results = data;
    renderSearchResults(data);
    updateResourceMetrics();
  }

  function renderSearchResults(data) {
    const rollupList = document.getElementById('searchRollupList');
    const matchesList = document.getElementById('searchMatchesList');

    if (rollupList) {
      rollupList.innerHTML = '';
      if (!data.folders || data.folders.length === 0) {
        rollupList.innerHTML = `<div class="empty-state-sm">No folders matched.</div>`;
      } else {
        data.folders.forEach((f) => {
          const div = document.createElement('div');
          div.className = 'rollup-item';
          div.innerHTML = `
            <span>📁 ${f.name}</span>
            <span class="rollup-hits-badge">${f.total_hits} hits</span>
          `;
          rollupList.appendChild(div);
        });
      }
    }

    if (matchesList) {
      matchesList.innerHTML = '';
      if (!data.matches || data.matches.length === 0) {
        matchesList.innerHTML = `<div class="empty-state-sm">No line matches found.</div>`;
      } else {
        data.matches.forEach((m) => {
          const div = document.createElement('div');
          div.style.marginBottom = '12px';
          div.innerHTML = `
            <div style="font-size: 11px; color: var(--accent-cyan); font-family: var(--font-mono); margin-bottom: 4px;">📂 ${m.path} : line ${m.line_number}</div>
            <div style="background-color: var(--bg-canvas); padding: 6px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; border: 1px solid var(--border-subtle);">${escapeHTML(m.line_content)}</div>
          `;
          matchesList.appendChild(div);
        });
      }
    }
  }

  // =========================================================================
  // Command Palette & Global Shortcuts
  // =========================================================================
  function initCommandPalette() {
    const modal = document.getElementById('commandPaletteModal');
    const input = document.getElementById('paletteInput');
    const closeBtn = document.getElementById('closePaletteBtn');
    const openBtn = document.getElementById('openCommandPaletteBtn');

    function openPalette() {
      modal?.classList.add('active');
      input?.focus();
      renderPaletteItems();
    }

    function closePalette() {
      modal?.classList.remove('active');
      if (input) input.value = '';
    }

    openBtn?.addEventListener('click', openPalette);
    closeBtn?.addEventListener('click', closePalette);

    modal?.addEventListener('click', (e) => {
      if (e.target === modal) closePalette();
    });

    input?.addEventListener('input', (e) => {
      renderPaletteItems(e.target.value.toLowerCase());
    });
  }

  const PALETTE_COMMANDS = [
    { title: 'Toggle Light / Dark Theme', action: () => document.getElementById('themeToggleBtn')?.click(), icon: '🌓' },
    { title: 'Switch to Review & Diffs', tab: 'review', icon: '🔍' },
    { title: 'Switch to Automation Flows', tab: 'flow', icon: '🌊' },
    { title: 'Switch to Search Explorer', tab: 'search', icon: '🔎' },
    { title: 'Switch to Environment Vault', tab: 'vault', icon: '🔐' },
    { title: 'Switch to System Diagnostics (Doctor)', tab: 'doctor', icon: '🩺' },
    { title: 'Switch to API & HTTP Diff', tab: 'httpdiff', icon: '🌐' },
    { title: 'Switch to Data File Viewer (Parquet/CSV)', tab: 'dataview', icon: '📊' },
    { title: 'Load Sample Parquet File', action: () => document.getElementById('dataLoadSampleBtn')?.click(), icon: '⚡' },
    { title: 'Compare Sample APIs', action: () => document.getElementById('httpLoadSampleBtn')?.click(), icon: '⚡' },
    { title: 'Toggle Side-by-Side Diff', action: () => document.getElementById('diffModeSplitBtn')?.click(), icon: '⚡' },
    { title: 'Refresh Workspace State', action: () => document.getElementById('refreshBtn')?.click(), icon: '🔄' },
  ];

  function renderPaletteItems(query = '') {
    const list = document.getElementById('paletteResults');
    if (!list) return;

    list.innerHTML = '';
    const filtered = PALETTE_COMMANDS.filter((c) =>
      c.title.toLowerCase().includes(query)
    );

    filtered.forEach((cmd) => {
      const div = document.createElement('div');
      div.className = 'palette-item';
      div.innerHTML = `<span>${cmd.icon} ${cmd.title}</span><kbd class="kbd-shortcut">Enter</kbd>`;
      div.addEventListener('click', () => {
        document.getElementById('commandPaletteModal')?.classList.remove('active');
        if (cmd.tab) switchTab(cmd.tab);
        if (cmd.action) cmd.action();
      });
      list.appendChild(div);
    });
  }

  function initShortcuts() {
    window.addEventListener('keydown', (e) => {
      // Cmd+K or Ctrl+K -> Command Palette
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        const modal = document.getElementById('commandPaletteModal');
        if (modal?.classList.contains('active')) {
          modal.classList.remove('active');
        } else {
          document.getElementById('openCommandPaletteBtn')?.click();
        }
      }

      // '[' or Cmd/Ctrl+B -> Toggle Sidebar
      if ((e.key === '[' || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'b')) && !['INPUT', 'TEXTAREA'].includes(e.target.tagName)) {
        e.preventDefault();
        document.getElementById('sidebarToggleBtn')?.click();
      }

      // 'f' or Shift+F when in dataview -> Toggle Zen full screen
      if ((e.key === 'f' || e.key === 'F') && !['INPUT', 'TEXTAREA'].includes(e.target.tagName)) {
        if (state.activeTab === 'dataview') {
          e.preventDefault();
          document.getElementById('gridZenToggleBtn')?.click();
        }
      }

      // Escape -> close palette or exit full-screen
      if (e.key === 'Escape') {
        document.getElementById('commandPaletteModal')?.classList.remove('active');
        const gridCard = document.getElementById('dataGridCard');
        if (gridCard?.classList.contains('full-screen-mode')) {
          gridCard.classList.remove('full-screen-mode');
          const zenText = document.getElementById('zenBtnText');
          if (zenText) zenText.textContent = 'Maximize';
          setTimeout(() => state.dataview?.tabulatorInstance?.redraw(true), 150);
        }
      }
    });
  }

  // =========================================================================
  // Process Resource Meter (CPU, Memory, Goroutines)
  // =========================================================================
  function initResourceMeter() {
    updateResourceMetrics();
    // Poll every 2 seconds for live CPU & RAM usage
    setInterval(updateResourceMetrics, 2000);
  }

  async function updateResourceMetrics() {
    const data = await fetchJSON('/api/metrics');
    if (!data) return;

    const cpuVal = document.getElementById('meterCpuVal');
    const ramVal = document.getElementById('meterRamVal');
    const workersVal = document.getElementById('meterWorkersVal');
    const dot = document.getElementById('meterWorkersDot');

    if (cpuVal) {
      cpuVal.textContent = `${data.cpu_percent.toFixed(1)}%`;
      if (data.cpu_percent > 70) {
        cpuVal.style.color = '#ef4444';
      } else if (data.cpu_percent > 30) {
        cpuVal.style.color = '#f59e0b';
      } else {
        cpuVal.style.color = '';
      }
    }

    if (ramVal) {
      ramVal.textContent = `${data.alloc_mb.toFixed(1)} MB`;
    }

    if (workersVal) {
      workersVal.textContent = `${data.goroutines} threads`;
    }

    if (dot) {
      if (data.cpu_percent > 40 || data.goroutines > 30) {
        dot.classList.add('high-load');
      } else {
        dot.classList.remove('high-load');
      }
    }
  }

  // =========================================================================
  // System Info & Toast Notifications
  // =========================================================================
  async function loadSystemInfo() {
    const data = await fetchJSON('/api/info');
    if (!data) return;

    state.workspace = { ...state.workspace, ...data };

    const wsNameEl = document.getElementById('workspaceName');
    const branchNameEl = document.getElementById('branchName');
    const aheadEl = document.getElementById('branchAheadCount');

    if (wsNameEl) wsNameEl.textContent = data.workspace_name || 'workspace';
    if (branchNameEl) branchNameEl.textContent = data.git_branch || 'main';
    if (aheadEl) aheadEl.textContent = `${data.ahead_count || 0} ahead`;
  }

  function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast';

    let icon = '⚡';
    if (type === 'error') icon = '🔴';
    if (type === 'success') icon = '🟢';
    if (type === 'warning') icon = '🟡';

    toast.innerHTML = `<span>${icon}</span><span>${escapeHTML(message)}</span>`;
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(8px)';
      toast.style.transition = 'all 0.2s ease';
      setTimeout(() => toast.remove(), 200);
    }, 3000);
  }

  // =========================================================================
  // Module 4: Environment Vault (.env.vault)
  // =========================================================================
  function initVaultView() {
    const envSelect = document.getElementById('vaultEnvSelect');
    envSelect?.addEventListener('change', (e) => {
      loadVaultData(e.target.value);
    });
  }

  async function loadVaultData(env = 'default') {
    const data = await fetchJSON(`/api/vault?env=${encodeURIComponent(env)}`);
    if (!data) return;

    const envSelect = document.getElementById('vaultEnvSelect');
    if (envSelect && data.environments) {
      envSelect.innerHTML = '';
      data.environments.forEach((envName) => {
        const opt = document.createElement('option');
        opt.value = envName;
        opt.textContent = `Environment: ${envName}`;
        if (envName === data.current_env) opt.selected = true;
        envSelect.appendChild(opt);
      });
    }

    const tbody = document.getElementById('vaultTableBody');
    if (!tbody) return;
    tbody.innerHTML = '';

    if (!data.variables || data.variables.length === 0) {
      tbody.innerHTML = `<tr><td colspan="3" class="empty-state-sm">No environment variables registered for this profile.</td></tr>`;
      return;
    }

    data.variables.forEach((v) => {
      const tr = document.createElement('tr');
      const isSecret = v.is_secret;
      const maskedVal = escapeHTML(v.masked || v.value || '••••••••');
      tr.innerHTML = `
        <td style="font-weight: 600; color: var(--accent);">${escapeHTML(v.key)}</td>
        <td class="secret-cell" data-masked="${maskedVal}" data-raw="${escapeHTML(v.value || maskedVal)}">
          <code>${maskedVal}</code>
        </td>
        <td>
          <button class="btn btn-sm btn-reveal-secret" ${!isSecret ? 'disabled' : ''}>
            ${isSecret ? 'Reveal' : 'Plain'}
          </button>
        </td>
      `;
      tbody.appendChild(tr);
    });

    tbody.querySelectorAll('.btn-reveal-secret').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const row = e.target.closest('tr');
        const cell = row?.querySelector('.secret-cell');
        const code = cell?.querySelector('code');
        if (!cell || !code) return;
        if (btn.textContent.trim() === 'Reveal') {
          code.textContent = cell.getAttribute('data-raw');
          code.style.color = 'var(--text-primary)';
          btn.textContent = 'Hide';
        } else {
          code.textContent = cell.getAttribute('data-masked');
          code.style.color = '';
          btn.textContent = 'Reveal';
        }
      });
    });
  }

  // =========================================================================
  // Module 5: System Health & Diagnostics (Doctor)
  // =========================================================================
  function initDoctorView() {
    // Initialized on tab switch
  }

  async function loadDoctorData() {
    const data = await fetchJSON('/api/doctor');
    if (!data) return;

    const container = document.getElementById('doctorContainer');
    if (!container) return;
    container.innerHTML = '';

    // Group checks by category
    const categories = {};
    (data.checks || []).forEach((c) => {
      if (!categories[c.category]) categories[c.category] = [];
      categories[c.category].push(c);
    });

    Object.entries(categories).forEach(([category, checks]) => {
      const card = document.createElement('div');
      card.className = 'doctor-card';

      let checksHTML = '';
      checks.forEach((c) => {
        let badgeClass = 'status-badge-pass';
        let icon = '✔';
        if (c.status === 'warn') {
          badgeClass = 'status-badge-warn';
          icon = '▲';
        } else if (c.status === 'fail') {
          badgeClass = 'status-badge-fail';
          icon = '✖';
        }

        checksHTML += `
          <div class="doctor-check-row">
            <span class="doctor-check-name">
              <span class="${badgeClass}">${icon}</span>
              ${escapeHTML(c.name)}
            </span>
            <span class="doctor-check-detail" title="${escapeHTML(c.detail)}">
              ${escapeHTML(c.detail)}
            </span>
          </div>
        `;
      });

      card.innerHTML = `
        <div class="doctor-card-title">
          <span>${escapeHTML(category)}</span>
          <span class="badge badge-success">${checks.length} verified</span>
        </div>
        ${checksHTML}
      `;
      container.appendChild(card);
    });
  }

  // =========================================================================
  // Module 6: Visual HTTP & API Response Diff Inspector
  // =========================================================================
  function initHTTPDiffView() {
    const runBtn = document.getElementById('httpRunDiffBtn');
    const loadSampleBtn = document.getElementById('httpLoadSampleBtn');
    const viewBadgesBtn = document.getElementById('httpViewBadgesBtn');
    const viewSplitBtn = document.getElementById('httpViewSplitBtn');
    const url1Input = document.getElementById('httpUrl1Input');
    const url2Input = document.getElementById('httpUrl2Input');

    runBtn?.addEventListener('click', () => {
      runHTTPDiff(url1Input?.value.trim(), url2Input?.value.trim());
    });

    loadSampleBtn?.addEventListener('click', () => {
      if (url1Input) url1Input.value = 'http://api.internal/v1/service';
      if (url2Input) url2Input.value = 'http://api.internal/v2/service';
      runHTTPDiff('http://api.internal/v1/service', 'http://api.internal/v2/service');
    });

    viewBadgesBtn?.addEventListener('click', () => {
      viewBadgesBtn.classList.add('active');
      viewSplitBtn?.classList.remove('active');
      const badgesPane = document.getElementById('httpViewBadgesPane');
      const splitPane = document.getElementById('httpViewSplitPane');
      if (badgesPane) badgesPane.style.display = 'flex';
      if (splitPane) splitPane.style.display = 'none';
      state.httpdiff.activeView = 'badges';
    });

    viewSplitBtn?.addEventListener('click', () => {
      viewSplitBtn.classList.add('active');
      viewBadgesBtn?.classList.remove('active');
      const badgesPane = document.getElementById('httpViewBadgesPane');
      const splitPane = document.getElementById('httpViewSplitPane');
      if (badgesPane) badgesPane.style.display = 'none';
      if (splitPane) splitPane.style.display = 'block';
      state.httpdiff.activeView = 'split';
    });

    [url1Input, url2Input].forEach((input) => {
      input?.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') runBtn?.click();
      });
    });
  }

  async function runHTTPDiff(url1 = '', url2 = '') {
    showToast('Comparing API endpoints...', 'info');
    const query = url1 && url2 ? `?url1=${encodeURIComponent(url1)}&url2=${encodeURIComponent(url2)}` : '';
    const data = await fetchJSON(`/api/http/diff${query}`);
    if (!data) return;

    state.httpdiff.results = data;
    renderHTTPDiffResults(data);
    showToast(`Diff computed: ${data.added_keys.length} added, ${data.removed_keys.length} removed, ${data.changed_values.length} changed`, 'info');
  }

  function renderHTTPDiffResults(data) {
    const statusEl = document.getElementById('diffStatusMatch');
    const latencyEl = document.getElementById('diffLatencyComp');
    const matchPctEl = document.getElementById('diffMatchPct');
    const keyCountsEl = document.getElementById('diffKeyCounts');

    if (statusEl) {
      statusEl.innerHTML = data.status_match
        ? `<span style="color:#10b981;">✔ [${data.status1}] Match</span>`
        : `<span style="color:#ef4444;">✖ [${data.status1}] vs [${data.status2}]</span>`;
    }

    if (latencyEl) {
      latencyEl.textContent = `${data.latency1} vs ${data.latency2} (${data.latency_delta})`;
    }

    if (matchPctEl) {
      matchPctEl.textContent = `${data.match_percentage.toFixed(1)}% Structural Match`;
    }

    if (keyCountsEl) {
      keyCountsEl.innerHTML = `
        <span style="color:#10b981;">+${data.added_keys.length}</span>
        <span style="color:#ef4444;">-${data.removed_keys.length}</span>
        <span style="color:#06b6d4;">~${data.changed_values.length}</span>
      `;
    }

    // Render Added Keys
    const addedList = document.getElementById('diffAddedList');
    if (addedList) {
      addedList.innerHTML = '';
      if (!data.added_keys || data.added_keys.length === 0) {
        addedList.innerHTML = `<div class="empty-state-sm">No added keys detected.</div>`;
      } else {
        data.added_keys.forEach((k) => {
          const div = document.createElement('div');
          div.className = 'diff-key-pill';
          div.innerHTML = `<span style="color:#10b981;">+ ${escapeHTML(k)}</span><span class="badge badge-success">Added</span>`;
          addedList.appendChild(div);
        });
      }
    }

    // Render Removed Keys
    const removedList = document.getElementById('diffRemovedList');
    if (removedList) {
      removedList.innerHTML = '';
      if (!data.removed_keys || data.removed_keys.length === 0) {
        removedList.innerHTML = `<div class="empty-state-sm">No removed keys detected.</div>`;
      } else {
        data.removed_keys.forEach((k) => {
          const div = document.createElement('div');
          div.className = 'diff-key-pill';
          div.innerHTML = `<span style="color:#ef4444;">- ${escapeHTML(k)}</span><span class="badge badge-danger">Removed</span>`;
          removedList.appendChild(div);
        });
      }
    }

    // Render Changed Values
    const changedList = document.getElementById('diffChangedList');
    if (changedList) {
      changedList.innerHTML = '';
      if (!data.changed_values || data.changed_values.length === 0) {
        changedList.innerHTML = `<div class="empty-state-sm">No value mismatches detected.</div>`;
      } else {
        data.changed_values.forEach((c) => {
          const div = document.createElement('div');
          div.className = 'diff-key-pill';
          div.innerHTML = `
            <span>
              <strong style="color:var(--text-primary);">${escapeHTML(c.path)}:</strong>
              <span style="color:#ef4444; text-decoration:line-through; margin:0 6px;">${escapeHTML(JSON.stringify(c.value1))}</span>
              <span style="color:#10b981;">${escapeHTML(JSON.stringify(c.value2))}</span>
            </span>
            <span class="badge badge-primary">Modified</span>
          `;
          changedList.appendChild(div);
        });
      }
    }

    // Render Split JSON Diff Table
    const splitBody = document.getElementById('httpSplitDiffBody');
    if (splitBody && data.split_rows) {
      splitBody.innerHTML = '';
      data.split_rows.forEach((row) => {
        const tr = document.createElement('tr');
        const leftCls = row.left_type === 'del' ? 'diff-line-del' : '';
        const rightCls = row.right_type === 'add' ? 'diff-line-add' : '';

        tr.innerHTML = `
          <td class="diff-line-num ${leftCls}">${row.left_num || ''}</td>
          <td class="diff-line-content ${leftCls}">${escapeHTML(row.left_content || '')}</td>
          <td class="diff-line-num ${rightCls}">${row.right_num || ''}</td>
          <td class="diff-line-content ${rightCls}">${escapeHTML(row.right_content || '')}</td>
        `;
        splitBody.appendChild(tr);
      });
    }
  }

  // =========================================================================
  // Module 7: Tabular Data File Viewer (Parquet, GZ, CSV, Excel)
  // =========================================================================
  function initDataView() {
    const dropzone = document.getElementById('dataDropzone');
    const fileInput = document.getElementById('dataFileInput');
    const uploadBtn = document.getElementById('dataUploadTriggerBtn');
    const loadSampleBtn = document.getElementById('dataLoadSampleBtn');
    const selectAllBtn = document.getElementById('dataSelectAllColsBtn');
    const resetColsBtn = document.getElementById('dataDeselectAllColsBtn');
    const filterInput = document.getElementById('dataFilterInput');
    const exportCsvBtn = document.getElementById('dataExportCsvBtn');
    const exportJsonBtn = document.getElementById('dataExportJsonBtn');
    const toggleColsBtn = document.getElementById('toggleColDrawerBtn');
    const zenBtn = document.getElementById('gridZenToggleBtn');
    const gridCard = document.getElementById('dataGridCard');
    const zenText = document.getElementById('zenBtnText');

    uploadBtn?.addEventListener('click', () => fileInput?.click());
    dropzone?.addEventListener('click', (e) => {
      if (e.target !== fileInput) fileInput?.click();
    });

    fileInput?.addEventListener('change', (e) => {
      const file = e.target.files?.[0];
      if (file) uploadDataFile(file);
    });

    dropzone?.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropzone.classList.add('drag-active');
    });

    dropzone?.addEventListener('dragleave', () => {
      dropzone.classList.remove('drag-active');
    });

    dropzone?.addEventListener('drop', (e) => {
      e.preventDefault();
      dropzone.classList.remove('drag-active');
      const file = e.dataTransfer.files?.[0];
      if (file) uploadDataFile(file);
    });

    loadSampleBtn?.addEventListener('click', () => {
      loadDataView('tests/fixtures/sample_metrics.parquet.gz');
    });

    selectAllBtn?.addEventListener('click', () => {
      if (!state.dataview.summary) return;
      const allCols = Array.isArray(state.dataview.summary?.columns) ? state.dataview.summary.columns : [];
      const colNames = allCols.map(c => c.Name || c.name);
      state.dataview.selectedColumns = [...colNames];

      document.querySelectorAll('#dataColumnsList .col-toggle-pill').forEach(pill => {
        pill.classList.add('active');
        const cb = pill.querySelector('input');
        if (cb) cb.checked = true;
      });

      if (state.dataview.tabulatorInstance) {
        colNames.forEach(c => state.dataview.tabulatorInstance.showColumn(c));
      }

      if (!state.dataview.isUploaded) {
        updateColumnProjection();
      }
    });

    resetColsBtn?.addEventListener('click', () => {
      if (!state.dataview.summary) return;
      const allCols = Array.isArray(state.dataview.summary?.columns) ? state.dataview.summary.columns : [];
      const firstFew = allCols.slice(0, 3).map(c => c.Name || c.name);
      state.dataview.selectedColumns = [...firstFew];

      document.querySelectorAll('#dataColumnsList .col-toggle-pill').forEach(pill => {
        const colName = pill.getAttribute('data-col-name');
        const isSelected = firstFew.includes(colName);
        pill.classList.toggle('active', isSelected);
        const cb = pill.querySelector('input');
        if (cb) cb.checked = isSelected;
      });

      if (state.dataview.tabulatorInstance) {
        allCols.forEach(c => {
          const colName = c.Name || c.name;
          if (firstFew.includes(colName)) {
            state.dataview.tabulatorInstance.showColumn(colName);
          } else {
            state.dataview.tabulatorInstance.hideColumn(colName);
          }
        });
      }

      if (!state.dataview.isUploaded) {
        updateColumnProjection();
      }
    });

    toggleColsBtn?.addEventListener('click', () => {
      const colsCard = document.getElementById('dataColumnsCard');
      if (colsCard) {
        const isHidden = colsCard.style.display === 'none';
        colsCard.style.display = isHidden ? 'block' : 'none';
        toggleColsBtn.textContent = isHidden ? 'Columns ▲' : 'Columns ▼';
      }
    });

    zenBtn?.addEventListener('click', () => {
      gridCard?.classList.toggle('full-screen-mode');
      const isZen = gridCard?.classList.contains('full-screen-mode');
      if (zenText) zenText.textContent = isZen ? 'Minimize' : 'Maximize';
      showToast(isZen ? 'Zen Mode (Press Esc to minimize)' : 'Exited Full Screen', 'info');
      setTimeout(() => state.dataview?.tabulatorInstance?.redraw(true), 150);
    });

    exportCsvBtn?.addEventListener('click', () => {
      if (state.dataview?.tabulatorInstance) {
        state.dataview.tabulatorInstance.download('csv', `${state.dataview.summary?.file_name || 'data'}.csv`);
      } else {
        exportDataViewRows('csv');
      }
    });

    exportJsonBtn?.addEventListener('click', () => {
      if (state.dataview?.tabulatorInstance) {
        state.dataview.tabulatorInstance.download('json', `${state.dataview.summary?.file_name || 'data'}.json`);
      } else {
        exportDataViewRows('json');
      }
    });

    // Row density selector
    document.querySelectorAll('.btn-data-density').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        document.querySelectorAll('.btn-data-density').forEach(b => b.classList.remove('active'));
        e.target.classList.add('active');
        const density = e.target.getAttribute('data-density');
        const container = document.getElementById('tabulatorGridContainer');
        if (container) {
          container.classList.remove('density-compact', 'density-spacious');
          if (density === 'compact') container.classList.add('density-compact');
          if (density === 'spacious') container.classList.add('density-spacious');
          state.dataview?.tabulatorInstance?.redraw(true);
        }
      });
    });

    // Global filter search box (attached once)
    filterInput?.addEventListener('input', (e) => {
      const query = e.target.value.toLowerCase();
      if (!state.dataview.tabulatorInstance) return;
      if (!query) {
        state.dataview.tabulatorInstance.clearFilter();
      } else {
        state.dataview.tabulatorInstance.setFilter((row) => {
          return Object.values(row).some(v => {
            if (v === null || v === undefined) return false;
            if (typeof v === 'object') {
              return JSON.stringify(v).toLowerCase().includes(query);
            }
            return String(v).toLowerCase().includes(query);
          });
        });
      }
    });
  }

  async function loadDataView(file = '', cols = [], filter = '') {
    showToast('Loading tabular data file...', 'info');
    let url = `/api/data/view?limit=100`;
    if (file) url += `&file=${encodeURIComponent(file)}`;
    if (cols.length > 0) url += `&cols=${encodeURIComponent(cols.join(','))}`;
    if (filter) url += `&filter=${encodeURIComponent(filter)}`;

    const data = await fetchJSON(url);
    if (!data) return;

    state.dataview.isUploaded = false;
    state.dataview.summary = data;
    const availableCols = Array.isArray(data?.columns) ? data.columns : [];
    if (cols.length === 0) {
      state.dataview.selectedColumns = (Array.isArray(data?.selected_columns) && data.selected_columns.length > 0)
        ? data.selected_columns
        : availableCols.map(c => c?.Name || c?.name);
    }
    renderDataView(data);
    showToast(`Loaded ${data?.file_name || 'file'} (${(data?.total_rows ?? 0).toLocaleString()} rows, ${data?.total_cols ?? 0} cols)`, 'info');
  }

  async function uploadDataFile(file) {
    showToast(`Uploading and inspecting ${file.name}...`, 'info');
    const formData = new FormData();
    formData.append('file', file);

    const token = getCSRFToken();
    const headers = {};
    if (token) {
      headers['X-CSRF-Token'] = token;
      headers['X-Shellix-Session'] = token;
    }

    try {
      const res = await fetch('/api/data/view?limit=100', {
        method: 'POST',
        headers,
        body: formData,
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      state.dataview.isUploaded = true;
      state.dataview.summary = data;
      const availableCols = Array.isArray(data?.columns) ? data.columns : [];
      state.dataview.selectedColumns = (Array.isArray(data?.selected_columns) && data.selected_columns.length > 0)
        ? data.selected_columns
        : availableCols.map(c => c?.Name || c?.name);
      renderDataView(data);
      showToast(`Successfully parsed ${file.name} (${(data?.format || '').toUpperCase()})`, 'success');
    } catch (err) {
      showToast(`Upload failed: ${err.message}`, 'error');
    }
  }

  function updateColumnProjection() {
    if (!state.dataview.summary) return;
    loadDataView(state.dataview.summary?.file_name || '', state.dataview.selectedColumns || [], state.dataview.filter || '');
  }

  function renderDataView(data) {
    if (!data) return;

    const dropzone = document.getElementById('dataDropzone');
    const profileBar = document.getElementById('dataProfileBar');
    const columnsCard = document.getElementById('dataColumnsCard');
    const gridCard = document.getElementById('dataGridCard');

    // Collapse dropzone to maximize screen space
    if (dropzone) dropzone.classList.add('collapsed');
    if (profileBar) profileBar.style.display = 'flex';
    if (columnsCard) columnsCard.style.display = 'block';
    if (gridCard) gridCard.style.display = 'block';

    const fnEl = document.getElementById('profileFileName');
    const fmtEl = document.getElementById('profileFormat');
    const rowsEl = document.getElementById('profileTotalRows');
    const colsEl = document.getElementById('profileColsCount');
    const sizeEl = document.getElementById('profileFileSize');

    if (fnEl) fnEl.textContent = data?.file_name || 'dataset';
    if (fmtEl) {
      const fmtStr = (data?.format || 'UNKNOWN').toUpperCase();
      const compStr = data?.compression || 'none';
      fmtEl.innerHTML = `<span class="badge badge-success">${escapeHTML(fmtStr)}</span> <span class="badge badge-warning">${escapeHTML(compStr)}</span>`;
    }
    const totalRows = data?.total_rows ?? 0;
    const totalCols = data?.total_cols ?? 0;
    if (rowsEl) rowsEl.textContent = `${totalRows.toLocaleString()} rows`;
    if (colsEl) colsEl.textContent = `${totalCols} columns`;
    if (sizeEl) sizeEl.textContent = formatBytes(data?.file_size || 0);

    const allCols = Array.isArray(data?.columns) ? data.columns : [];
    const rows = Array.isArray(data?.rows) ? data.rows : [];

    // Fallback: if columns array was empty but rows exist, extract columns from first row
    let effectiveCols = allCols;
    if (effectiveCols.length === 0 && rows.length > 0) {
      effectiveCols = Object.keys(rows[0]).map(k => ({ name: k, type: 'UNKNOWN', nullable: true }));
      if (state.dataview.summary && (!Array.isArray(state.dataview.summary.columns) || state.dataview.summary.columns.length === 0)) {
        state.dataview.summary.columns = effectiveCols;
      }
    }

    if (!Array.isArray(state.dataview.selectedColumns) || state.dataview.selectedColumns.length === 0) {
      state.dataview.selectedColumns = (Array.isArray(data?.selected_columns) && data.selected_columns.length > 0)
        ? data.selected_columns
        : effectiveCols.map(c => c?.Name || c?.name);
    }

    // Render Column Toggle Pills
    const colsList = document.getElementById('dataColumnsList');
    if (colsList) {
      colsList.innerHTML = '';
      effectiveCols.forEach((col) => {
        const colName = col.Name || col.name;
        const colType = col.Type || col.type || 'STRING';
        const isSelected = state.dataview.selectedColumns.includes(colName);

        const label = document.createElement('label');
        label.className = `col-toggle-pill ${isSelected ? 'active' : ''}`;
        label.setAttribute('data-col-name', colName);
        label.innerHTML = `
          <input type="checkbox" ${isSelected ? 'checked' : ''}>
          <span>${escapeHTML(colName)}</span>
          <span style="opacity: 0.6; font-size: 10px;">${escapeHTML(colType)}</span>
        `;
        const cb = label.querySelector('input');
        cb?.addEventListener('change', (e) => {
          const isChecked = e.target.checked;
          if (isChecked) {
            if (!state.dataview.selectedColumns.includes(colName)) {
              state.dataview.selectedColumns.push(colName);
            }
            if (state.dataview.tabulatorInstance) {
              state.dataview.tabulatorInstance.showColumn(colName);
            }
          } else {
            state.dataview.selectedColumns = state.dataview.selectedColumns.filter(c => c !== colName);
            if (state.dataview.tabulatorInstance) {
              state.dataview.tabulatorInstance.hideColumn(colName);
            }
          }
          label.classList.toggle('active', isChecked);
          if (!state.dataview.isUploaded) {
            updateColumnProjection();
          }
        });
        colsList.appendChild(label);
      });
    }

    const statsEl = document.getElementById('dataGridStats');
    if (statsEl) {
      statsEl.textContent = `Displaying ${rows.length} of ${totalRows.toLocaleString()} rows (Virtual DOM 60 FPS)`;
    }

    // Build Tabulator columns with interactive header filter inputs & freeze first column
    const tabColumns = effectiveCols.map((col, idx) => {
      const colName = col.Name || col.name;
      const isVisible = state.dataview.selectedColumns.includes(colName);
      return {
        title: colName,
        field: colName,
        visible: isVisible,
        headerFilter: "input",
        headerFilterPlaceholder: "Filter...",
        resizable: true,
        frozen: idx === 0, // Freeze first column for horizontal scrolling
        sorter: "string",
        formatter: function(cell) {
          let val = cell.getValue();
          if (val === undefined || val === null) {
            const field = cell.getColumn().getField();
            if (field && field.includes('.')) {
              const parts = field.split('.');
              let cur = cell.getData();
              for (const p of parts) {
                if (cur === null || cur === undefined) {
                  cur = undefined;
                  break;
                }
                cur = cur[p];
              }
              if (cur !== undefined) {
                val = cur;
              }
            }
          }
          if (val !== null && val !== undefined && typeof val === 'object') {
            return JSON.stringify(val);
          }
          return val ?? '';
        },
      };
    });

    if (state.dataview.tabulatorInstance) {
      state.dataview.tabulatorInstance.setColumns(tabColumns);
      state.dataview.tabulatorInstance.setData(rows);
    } else if (window.Tabulator) {
      state.dataview.tabulatorInstance = new Tabulator("#tabulatorGridContainer", {
        nestedFieldSeparator: false,
        data: rows,
        columns: tabColumns,
        layout: "fitDataFill",
        movableColumns: true,
        resizableColumns: true,
        renderVertical: "virtual",
        height: "520px",
        placeholder: "No records found matching criteria",
      });
    }
  }

  function exportDataViewRows(format) {
    if (!state.dataview.summary || !Array.isArray(state.dataview.summary.rows)) {
      showToast('No rows to export', 'warning');
      return;
    }
    const data = state.dataview.summary.rows;
    if (format === 'csv') {
      const headers = Array.isArray(state.dataview.selectedColumns) && state.dataview.selectedColumns.length > 0
        ? state.dataview.selectedColumns
        : (data.length > 0 ? Object.keys(data[0]) : []);
      const csvLines = [headers.join(',')];
      data.forEach((row) => {
        const vals = headers.map(h => {
          let val = row[h];
          if ((val === undefined || val === null) && h.includes('.')) {
            const parts = h.split('.');
            let cur = row;
            for (const p of parts) {
              if (cur === null || cur === undefined) {
                cur = undefined;
                break;
              }
              cur = cur[p];
            }
            if (cur !== undefined) val = cur;
          }
          if (val !== null && val !== undefined && typeof val === 'object') {
            val = JSON.stringify(val);
          }
          return `"${String(val ?? '').replace(/"/g, '""')}"`;
        });
        csvLines.push(vals.join(','));
      });
      downloadBlob(csvLines.join('\n'), `${state.dataview.summary?.file_name || 'data'}.csv`, 'text/csv');
      showToast('Exported CSV file', 'info');
    } else {
      const jsonStr = JSON.stringify(data, null, 2);
      downloadBlob(jsonStr, `${state.dataview.summary?.file_name || 'data'}.json`, 'application/json');
      showToast('Exported JSON file', 'info');
    }
  }

  function formatBytes(bytes) {
    if (!bytes || bytes < 1024) return `${bytes || 0} B`;
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  }

  function escapeHTML(str) {
    if (!str) return '';
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
})();
