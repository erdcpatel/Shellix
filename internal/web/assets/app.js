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
    grid: {
      cachedData: [],      // In-memory cache
      filteredData: [],
      currentPage: 1,
      pageSize: 12,
      sortColumn: null,
      sortAsc: true,
      searchQuery: '',
    },
    flows: [],
    activeFlow: null,
    search: {
      query: '',
      ext: '',
      results: null,
      isSearching: false,
    },
  };

  // =========================================================================
  // API Client Helpers
  // =========================================================================
  async function fetchJSON(url, options = {}) {
    try {
      const res = await fetch(url, options);
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
    initGridView();
    initFlowView();
    initSearchView();
    initVaultView();
    initDoctorView();
    initCommandPalette();

    // Initial data hydration
    loadSystemInfo();
    loadGitStatus();
    loadFlows();
    loadGridSampleData();

    // Check initial hash route
    const hash = window.location.hash.replace('#', '');
    if (hash && ['review', 'grid', 'flow', 'search', 'vault', 'doctor'].includes(hash)) {
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

    document.getElementById('refreshBtn')?.addEventListener('click', () => {
      showToast('Refreshing workspace state...', 'info');
      loadSystemInfo();
      loadGitStatus();
      if (state.activeTab === 'grid') loadGridSampleData(true);
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
    if (tabId === 'grid' && state.grid.cachedData.length === 0) {
      loadGridSampleData();
    } else if (tabId === 'vault') {
      loadVaultData();
    } else if (tabId === 'doctor') {
      loadDoctorData();
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

  // =========================================================================
  // Module 2: Data Grid & Tables (with in-memory client-side cache)
  // =========================================================================
  function initGridView() {
    const searchInput = document.getElementById('gridSearchInput');
    const refreshBtn = document.getElementById('gridRefreshDataBtn');
    const prevBtn = document.getElementById('gridPrevPageBtn');
    const nextBtn = document.getElementById('gridNextPageBtn');

    searchInput?.addEventListener('input', (e) => {
      state.grid.searchQuery = e.target.value.toLowerCase();
      state.grid.currentPage = 1;
      filterAndRenderGrid();
    });

    refreshBtn?.addEventListener('click', () => {
      loadGridSampleData(true);
    });

    prevBtn?.addEventListener('click', () => {
      if (state.grid.currentPage > 1) {
        state.grid.currentPage--;
        renderGridRows();
      }
    });

    nextBtn?.addEventListener('click', () => {
      const maxPage = Math.ceil(state.grid.filteredData.length / state.grid.pageSize);
      if (state.grid.currentPage < maxPage) {
        state.grid.currentPage++;
        renderGridRows();
      }
    });

    // Row density toggler
    document.querySelectorAll('.btn-density').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        document.querySelectorAll('.btn-density').forEach((b) => b.classList.remove('active'));
        e.target.classList.add('active');
        const density = e.target.getAttribute('data-density');
        const table = document.getElementById('dataTable');
        if (table) {
          table.classList.remove('density-compact', 'density-spacious');
          if (density === 'compact') table.classList.add('density-compact');
          if (density === 'spacious') table.classList.add('density-spacious');
        }
      });
    });

    // Grid export controls
    document.getElementById('exportCsvBtn')?.addEventListener('click', () => {
      exportGridData('csv');
    });
    document.getElementById('exportJsonBtn')?.addEventListener('click', () => {
      exportGridData('json');
    });
  }

  function exportGridData(format) {
    const data = state.grid.filteredData;
    if (!data || data.length === 0) {
      showToast('No records to export', 'warning');
      return;
    }

    if (format === 'csv') {
      const headers = Object.keys(data[0]);
      const csvRows = [headers.join(',')];
      data.forEach((row) => {
        const values = headers.map((h) => {
          const val = row[h] === undefined ? '' : String(row[h]);
          return `"${val.replace(/"/g, '""')}"`;
        });
        csvRows.push(values.join(','));
      });
      downloadBlob(csvRows.join('\n'), 'shellix-data.csv', 'text/csv');
      showToast(`Exported ${data.length} records to CSV`, 'info');
    } else {
      const jsonStr = JSON.stringify(data, null, 2);
      downloadBlob(jsonStr, 'shellix-data.json', 'application/json');
      showToast(`Exported ${data.length} records to JSON`, 'info');
    }
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

  async function loadGridSampleData(force = false) {
    if (!force && state.grid.cachedData.length > 0) {
      filterAndRenderGrid();
      return;
    }

    const data = await fetchJSON('/api/data/sample');
    if (!data) return;

    state.grid.cachedData = data.records || [];
    filterAndRenderGrid();
    showToast(`Loaded ${state.grid.cachedData.length} grid records into memory`, 'info');
  }

  function filterAndRenderGrid() {
    const q = state.grid.searchQuery;
    if (!q) {
      state.grid.filteredData = [...state.grid.cachedData];
    } else {
      state.grid.filteredData = state.grid.cachedData.filter((row) =>
        Object.values(row).some((val) =>
          String(val).toLowerCase().includes(q)
        )
      );
    }

    // Sort if active
    if (state.grid.sortColumn) {
      const col = state.grid.sortColumn;
      const asc = state.grid.sortAsc ? 1 : -1;
      state.grid.filteredData.sort((a, b) => {
        if (a[col] < b[col]) return -1 * asc;
        if (a[col] > b[col]) return 1 * asc;
        return 0;
      });
    }

    renderGridHeader();
    renderGridRows();
  }

  function renderGridHeader() {
    const headerRow = document.getElementById('gridTableHeader');
    if (!headerRow || state.grid.cachedData.length === 0) return;

    const sample = state.grid.cachedData[0];
    const columns = Object.keys(sample);

    headerRow.innerHTML = '';
    columns.forEach((col) => {
      const th = document.createElement('th');
      const isSorted = state.grid.sortColumn === col;
      const sortArrow = isSorted ? (state.grid.sortAsc ? ' ▲' : ' ▼') : '';

      th.textContent = col.toUpperCase() + sortArrow;
      th.addEventListener('click', () => {
        if (state.grid.sortColumn === col) {
          state.grid.sortAsc = !state.grid.sortAsc;
        } else {
          state.grid.sortColumn = col;
          state.grid.sortAsc = true;
        }
        filterAndRenderGrid();
      });
      headerRow.appendChild(th);
    });
  }

  function renderGridRows() {
    const tbody = document.getElementById('gridTableBody');
    const statsEl = document.getElementById('gridRecordStats');
    const pageInd = document.getElementById('gridPageIndicator');
    if (!tbody) return;

    const total = state.grid.filteredData.length;
    const maxPage = Math.max(1, Math.ceil(total / state.grid.pageSize));

    if (statsEl) {
      statsEl.textContent = `Showing ${total} records (Cached in memory)`;
    }
    if (pageInd) {
      pageInd.textContent = `Page ${state.grid.currentPage} of ${maxPage}`;
    }

    tbody.innerHTML = '';

    if (total === 0) {
      tbody.innerHTML = `<tr><td colspan="6" class="empty-state-sm">No matching records found in cache.</td></tr>`;
      return;
    }

    const start = (state.grid.currentPage - 1) * state.grid.pageSize;
    const pageRows = state.grid.filteredData.slice(start, start + state.grid.pageSize);

    pageRows.forEach((row) => {
      const tr = document.createElement('tr');
      Object.values(row).forEach((val) => {
        const td = document.createElement('td');
        td.textContent = String(val);
        tr.appendChild(td);
      });
      tbody.appendChild(tr);
    });
  }

  // =========================================================================
  // Module 3: Automation Flows View
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
  // Module 4: Search Explorer View
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
    { title: 'Switch to Data Grid & Tables', tab: 'grid', icon: '📊' },
    { title: 'Switch to Automation Flows', tab: 'flow', icon: '🌊' },
    { title: 'Switch to Search Explorer', tab: 'search', icon: '🔎' },
    { title: 'Switch to Environment Vault', tab: 'vault', icon: '🔐' },
    { title: 'Switch to System Diagnostics (Doctor)', tab: 'doctor', icon: '🩺' },
    { title: 'Export Data Grid to CSV', action: () => document.getElementById('exportCsvBtn')?.click(), icon: '📥' },
    { title: 'Export Data Grid to JSON', action: () => document.getElementById('exportJsonBtn')?.click(), icon: '📄' },
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

      // Escape -> close palette
      if (e.key === 'Escape') {
        document.getElementById('commandPaletteModal')?.classList.remove('active');
      }
    });
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
  // Module 5: Environment Vault (.env.vault)
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
  // Module 6: System Health & Diagnostics (Doctor)
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
