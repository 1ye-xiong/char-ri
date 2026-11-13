(function () {
  const FL = window.FL;
  const state = {
    tab: 'applications',
    editingId: '',
    applications: [],
    vehicles: [],
    activity: null,
    timer: null
  };

  document.addEventListener('DOMContentLoaded', init);

  function init() {
    bindEvents();
    if (FL.token && FL.profile) {
      showApp();
      refreshAll();
    } else {
      showLogin('');
    }
  }

  function bindEvents() {
    document.getElementById('loginForm').addEventListener('submit', onLogin);
    document.getElementById('loginAccount').addEventListener('change', function () {
      document.getElementById('loginPassword').value = defaultPassword(this.value);
    });
    document.querySelectorAll('.tab').forEach(function (button) {
      button.addEventListener('click', function () { switchTab(button.dataset.tab); });
    });
    document.getElementById('saveDraftBtn').addEventListener('click', function () { saveApplication(false); });
    document.getElementById('saveSubmitBtn').addEventListener('click', function () { saveApplication(true); });
    document.getElementById('cancelEditBtn').addEventListener('click', resetApplicationForm);
    document.getElementById('refreshApplications').addEventListener('click', function () { guard(loadApplications); });
    document.getElementById('refreshApprovals').addEventListener('click', function () { guard(loadApplications); });
    document.getElementById('refreshCreatePanel').addEventListener('click', function () { guard(loadApplications); });
    document.getElementById('createActivityBtn').addEventListener('click', createActivity);
    document.getElementById('refreshActivity').addEventListener('click', function () { guard(loadActivity); });
    document.getElementById('refreshHistory').addEventListener('click', function () { guard(loadHistory); });
    document.getElementById('refreshVehicles').addEventListener('click', function () { guard(loadVehicles); });
    document.getElementById('refreshAudits').addEventListener('click', function () { guard(loadAudits); });
  }

  function defaultPassword(account) {
    return { rd_wang: 'rd123456', lead_li: 'lead123456', ops_zhao: 'ops123456' }[account] || '';
  }

  async function onLogin(event) {
    event.preventDefault();
    const account = document.getElementById('loginAccount').value;
    const password = document.getElementById('loginPassword').value;
    try {
      const data = await FL.api('/login', { method: 'POST', body: { account: account, password: password } });
      FL.saveSession(data);
      showApp();
      refreshAll();
      FL.toast('登录成功：' + FL.roleName(data.role), 'ok');
    } catch (err) {
      document.getElementById('loginHint').textContent = err.message;
    }
  }

  function logout() {
    FL.clearSession();
    stopTimer();
    document.getElementById('appPanel').classList.add('hidden');
    showLogin('');
    FL.toast('已退出登录');
  }

  function showLogin(hint) {
    document.getElementById('loginPanel').classList.remove('hidden');
    document.getElementById('loginHint').textContent = hint || '';
    document.querySelectorAll('.tab-panel').forEach(function (panel) { panel.classList.add('hidden'); });
  }

  function showApp() {
    document.getElementById('loginPanel').classList.add('hidden');
    document.getElementById('appPanel').classList.remove('hidden');
    renderSession();
    const role = FL.profile.role;
    document.querySelectorAll('.tab').forEach(function (button) {
      const visible = {
        applications: role === 'R',
        approvals: role === 'A',
        activity: role === 'A' || role === 'O',
        vehicles: role === 'O',
        audits: true
      }[button.dataset.tab];
      button.classList.toggle('hidden', !visible);
    });
    switchTab(role === 'R' ? 'applications' : (role === 'A' ? 'approvals' : 'activity'));
  }

  function renderSession() {
    const profile = FL.profile || {};
    document.getElementById('sessionBox').innerHTML =
      '<span class="badge blue">' + FL.escape(profile.displayName || profile.userId) + '</span>' +
      '<span class="badge gray">' + FL.escape(FL.roleName(profile.role)) + '</span>' +
      '<span class="muted">超时判定 ' + FL.escape(profile.timeoutSeconds || 60) + ' 秒</span>' +
      '<button class="btn small ghost" id="logoutBtn">退出</button>';
    document.getElementById('logoutBtn').addEventListener('click', logout);
  }

  function switchTab(tab) {
    state.tab = tab;
    document.querySelectorAll('.tab').forEach(function (button) {
      button.classList.toggle('active', button.dataset.tab === tab);
    });
    document.querySelectorAll('.tab-panel').forEach(function (panel) {
      panel.classList.toggle('hidden', panel.id !== 'tab-' + tab);
    });
    if (tab === 'activity') {
      startTimer();
      guard(loadActivity);
      guard(loadHistory);
    } else {
      stopTimer();
    }
    if (tab === 'vehicles') guard(loadVehicles);
    if (tab === 'audits') guard(loadAudits);
  }

  function startTimer() {
    stopTimer();
    state.timer = setInterval(function () {
      if (state.tab === 'activity') loadActivity(true);
    }, 5000);
  }

  function stopTimer() {
    if (state.timer) {
      clearInterval(state.timer);
      state.timer = null;
    }
  }

  async function refreshAll() {
    await guard(loadApplications);
    await guard(loadVehicles);
    await guard(loadActivity);
    await guard(loadHistory);
    if (state.tab === 'audits') await guard(loadAudits);
  }

  async function guard(fn, silent) {
    try {
      return await fn(silent);
    } catch (err) {
      if (err.status === 401) {
        FL.clearSession();
        stopTimer();
        document.getElementById('appPanel').classList.add('hidden');
        showLogin('登录已失效，请重新登录：' + err.message);
        return null;
      }
      if (!silent) FL.toast(err.message || '请求失败', 'err');
      return null;
    }
  }

  async function loadApplications() {
    const data = await FL.api('/applications');
    state.applications = (data && data.items) || [];
    renderApplications();
    renderApprovals();
    renderCreatePanel();
  }

  function renderApplications() {
    const body = document.querySelector('#applicationTable tbody');
    body.innerHTML = '';
    const history = document.getElementById('applicationHistory');
    history.innerHTML = '';
    if (state.applications.length === 0) {
      body.innerHTML = '<tr><td colspan="8" class="empty">暂无申请，先在上方创建一个草稿。</td></tr>';
      return;
    }
    state.applications.forEach(function (app) {
      const actions = [];
      if (app.can_submit) {
        actions.push('<button class="btn small primary" data-submit="' + FL.escape(app.id) + '">' + (app.status === 'rejected' ? '修改后重提' : '提交审批') + '</button>');
      }
      if (app.can_edit) {
        actions.push('<button class="btn small" data-edit="' + FL.escape(app.id) + '">修改</button>');
      }
      if (actions.length === 0) actions.push('<span class="muted">只读</span>');
      const row = document.createElement('tr');
      row.innerHTML =
        '<td>v' + app.version_no + '</td>' +
        '<td>' + FL.escape(app.reason) + '</td>' +
        '<td>' + app.interval_seconds + ' 秒</td>' +
        '<td>' + FL.badge(app.status) + '</td>' +
        '<td>' + FL.escape(app.submitted_at || '-') + '</td>' +
        '<td>' + FL.escape(app.decided_by || '-') + '</td>' +
        '<td>' + FL.escape(app.reject_reason || '-') + '</td>' +
        '<td>' + actions.join(' ') + '</td>';
      body.appendChild(row);
      if (app.history && app.history.length > 0) {
        app.history.forEach(function (version) {
          const item = document.createElement('div');
          item.className = 'history-item';
          item.textContent = '历史 v' + version.version_no + ' · ' + FL.statusText(version.status) +
            ' · ' + version.interval_seconds + ' 秒 · ' + version.reason +
            (version.reject_reason ? ' · 退回原因：' + version.reject_reason : '') +
            ' · ' + (version.created_at || '');
          history.appendChild(item);
        });
      }
    });
    body.querySelectorAll('[data-submit]').forEach(function (button) {
      button.addEventListener('click', function () { guard(function () { return submitApplication(button.dataset.submit); }); });
    });
    body.querySelectorAll('[data-edit]').forEach(function (button) {
      button.addEventListener('click', function () { fillApplicationForm(button.dataset.edit); });
    });
  }

  function fillApplicationForm(id) {
    const app = state.applications.find(function (item) { return item.id === id; });
    if (!app) return;
    state.editingId = app.id;
    document.getElementById('appReason').value = app.reason;
    document.getElementById('appInterval').value = app.interval_seconds || 10;
    document.getElementById('cancelEditBtn').classList.remove('hidden');
    document.getElementById('applicationHint').textContent = '正在修改 v' + app.version_no + '（' + FL.statusText(app.status) + '）';
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function resetApplicationForm() {
    state.editingId = '';
    document.getElementById('appReason').value = '';
    document.getElementById('appInterval').value = 10;
    document.getElementById('cancelEditBtn').classList.add('hidden');
    document.getElementById('applicationHint').textContent = '';
  }

  async function saveApplication(submit) {
    const reason = document.getElementById('appReason').value.trim();
    const interval = Number(document.getElementById('appInterval').value);
    const hint = document.getElementById('applicationHint');
    hint.textContent = '';
    try {
      let app;
      if (state.editingId) {
        app = await FL.api('/applications/' + state.editingId, { method: 'PUT', body: { reason: reason, interval_seconds: interval } });
      } else {
        app = await FL.api('/applications', { method: 'POST', body: { reason: reason, interval_seconds: interval } });
      }
      if (submit) {
        await FL.api('/applications/' + app.id + '/submit', { method: 'POST' });
        FL.toast('已提交审批', 'ok');
      } else {
        FL.toast('草稿已保存', 'ok');
      }
      resetApplicationForm();
      await loadApplications();
    } catch (err) {
      hint.textContent = err.message;
      FL.toast(err.message, 'err');
    }
  }

  async function submitApplication(id) {
    await FL.api('/applications/' + id + '/submit', { method: 'POST' });
    FL.toast('已提交审批', 'ok');
    await loadApplications();
  }

  function renderApprovals() {
    const body = document.querySelector('#approvalTable tbody');
    body.innerHTML = '';
    const pending = state.applications.filter(function (app) {
      return app.status === 'pending' && app.can_decide;
    });
    if (pending.length === 0) {
      body.innerHTML = '<tr><td colspan="6" class="empty">没有需要你处理的待审批申请。</td></tr>';
      return;
    }
    pending.forEach(function (app) {
      const row = document.createElement('tr');
      row.innerHTML =
        '<td>' + FL.escape(app.requester_name || app.requester_id) + '</td>' +
        '<td>v' + app.version_no + '</td>' +
        '<td>' + FL.escape(app.reason) + '</td>' +
        '<td>' + app.interval_seconds + ' 秒</td>' +
        '<td>' + FL.escape(app.submitted_at || '-') + '</td>' +
        '<td><button class="btn small primary" data-approve="' + FL.escape(app.id) + '">批准</button>' +
        ' <button class="btn small danger" data-reject="' + FL.escape(app.id) + '">退回</button></td>';
      body.appendChild(row);
    });
    body.querySelectorAll('[data-approve]').forEach(function (button) {
      button.addEventListener('click', function () { guard(function () { return decideApplication(button.dataset.approve, true); }); });
    });
    body.querySelectorAll('[data-reject]').forEach(function (button) {
      button.addEventListener('click', function () { guard(function () { return decideApplication(button.dataset.reject, false); }); });
    });
  }

  async function decideApplication(id, approve) {
    if (approve) {
      await FL.api('/applications/' + id + '/approve', { method: 'POST' });
      FL.toast('已批准，参数锁定', 'ok');
    } else {
      const reason = window.prompt('请输入退回原因（必填，≤200 字）');
      if (!reason) return;
      await FL.api('/applications/' + id + '/reject', { method: 'POST', body: { reason: reason } });
      FL.toast('已退回申请人', 'ok');
    }
    await loadApplications();
  }

  function renderCreatePanel() {
    const card = document.getElementById('createActivityCard');
    const approved = state.applications.filter(function (app) { return app.status === 'approved'; });
    card.classList.toggle('hidden', approved.length === 0 || (state.activity && state.activity.activity));
    const select = document.getElementById('createApplicationSelect');
    select.innerHTML = approved.map(function (app) {
      return '<option value="' + FL.escape(app.id) + '">v' + app.version_no + ' · ' + app.interval_seconds + ' 秒 · ' + FL.escape(app.reason) + '</option>';
    }).join('');
    const options = document.getElementById('pilotOptions');
    options.innerHTML = state.vehicles.map(function (vehicle) {
      return '<label><input type="checkbox" value="' + FL.escape(vehicle.id) + '"> ' + FL.escape(vehicle.id) +
        '（' + (vehicle.online ? '在线' : '离线') + '）</label>';
    }).join('');
  }

  async function createActivity() {
    const applicationId = document.getElementById('createApplicationSelect').value;
    const pilots = Array.from(document.querySelectorAll('#pilotOptions input:checked')).map(function (input) { return input.value; });
    const hint = document.getElementById('createActivityHint');
    hint.textContent = '';
    if (pilots.length !== 2) {
      hint.textContent = '必须选择 2 台不同的试点车';
      return;
    }
    try {
      state.activity = await FL.api('/activities', { method: 'POST', body: { application_id: applicationId, pilot_vehicles: pilots } });
      FL.toast('活动已创建，状态：待试点', 'ok');
      await loadApplications();
      await loadActivity();
      await loadHistory();
    } catch (err) {
      hint.textContent = err.message;
      FL.toast(err.message, 'err');
    }
  }

  async function loadActivity(silent) {
    const detail = await FL.api('/current-activity');
    state.activity = detail && detail.activity ? detail : null;
    renderActivity();
    if (!silent) renderCreatePanel();
  }

  async function openActivity(id) {
    try {
      state.activity = await FL.api('/activities/' + id);
      renderActivity();
      FL.toast('已加载活动 ' + id.slice(0, 8), 'ok');
    } catch (err) {
      FL.toast(err.message, 'err');
    }
  }

  function renderActivity() {
    const empty = document.getElementById('activityEmpty');
    const detail = document.getElementById('activityDetail');
    if (!state.activity) {
      empty.classList.remove('hidden');
      detail.classList.add('hidden');
      return;
    }
    empty.classList.add('hidden');
    detail.classList.remove('hidden');
    const activity = state.activity.activity;
    const actions = state.activity.actions || {};
    const summary = state.activity.summary || {};
    document.getElementById('serverClock').textContent =
      '服务端时间 ' + (state.activity.server_time || '-') + ' · 超时阈值 ' + (state.activity.timeout_seconds || 60) + ' 秒';

    const actionButtons = [
      ['start_pilot', '开始试点', actions.can_start_pilot, ''],
      ['promote', '确认推广', actions.can_promote, actions.promote_blocked_reason || ''],
      ['stop', '停止活动', actions.can_stop, ''],
      ['approve_restore', '批准恢复', actions.can_approve_restore, ''],
      ['start_restore', '发起恢复下发', actions.can_start_restore, ''],
      ['complete', '标记活动完成', actions.can_mark_completed, ''],
      ['restored', '标记恢复完成', actions.can_mark_restored, '']
    ].map(function (item) {
      return '<button class="btn small ' + (item[0] === 'stop' ? 'danger' : 'primary') + '" data-action="' + item[0] + '"' +
        (item[2] ? '' : ' disabled title="' + FL.escape(item[3] || '当前状态或角色不可用') + '"') + '>' + item[1] + '</button>';
    }).join('');

    const meta = [];
    meta.push('活动 ' + FL.escape(activity.id));
    meta.push('申请版本 ' + (state.activity.application ? 'v' + state.activity.application.version_no : '-'));
    meta.push('间隔 ' + activity.interval_seconds + ' 秒');
    meta.push('试点 ' + (activity.pilot_vehicles || []).join('、'));
    meta.push('推广 ' + (activity.promotion_vehicles || []).join('、'));
    if (activity.created_at) meta.push('创建于 ' + activity.created_at);
    if (activity.stopped_at) meta.push('停止人 ' + (activity.stopped_by || '-') + ' / ' + activity.stopped_at);
    if (activity.restore_approved_at) meta.push('恢复批准 ' + activity.restore_approved_by + ' / ' + activity.restore_approved_at);
    if (activity.completed_at) meta.push('完成标记 ' + activity.completed_by + ' / ' + activity.completed_at);
    if (activity.restored_at) meta.push('恢复完成 ' + activity.restored_by + ' / ' + activity.restored_at);

    let scopeHtml = '';
    if (activity.stop_scope) {
      try {
        const scope = JSON.parse(activity.stop_scope);
        scopeHtml = '<div class="row"><strong>停止范围快照</strong><span class="muted">停止只阻止未发布部分，不撤销车端已生效配置；需要恢复请走恢复流程。</span></div>' +
          '<div class="row"><span>已发：' + ((scope.released || []).map(function (item) {
            return FL.escape(item.vehicle_id) + '(v' + item.version + ' ' + FL.statusText(item.status) + ')';
          }).join('，') || '无') + '</span></div>' +
          '<div class="row"><span>未发：' + ((scope.not_released || []).join('、') || '无') + '</span></div>';
      } catch (err) {
        scopeHtml = '<pre class="scope">' + FL.escape(activity.stop_scope) + '</pre>';
      }
    }

    const rows = (state.activity.rows || []).map(function (row) {
      return '<tr>' +
        '<td><strong>' + FL.escape(row.vehicle_id) + '</strong></td>' +
        '<td>' + FL.escape(FL.batchText(row.batch)) + '</td>' +
        '<td>' + (row.released ? 'v' + row.version : '-') + '</td>' +
        '<td>' + row.baseline_interval + ' 秒</td>' +
        '<td>' + row.expected_interval + ' 秒</td>' +
        '<td>' + row.effective_interval + ' 秒（v' + row.effective_version + '）</td>' +
        '<td>' + FL.badge(row.status) + '</td>' +
        '<td>' + FL.escape(row.failure_reason || '-') + '</td>' +
        '<td>' + FL.escape(row.dispatched_at || '-') + '</td>' +
        '<td>' + FL.escape(row.receipt_at || '-') + '</td>' +
        '<td>' + FL.escape(row.timeout_at || '-') + (row.late ? '（迟到成功）' : '') + '</td>' +
        '<td>' + (row.duplicate_receipts || 0) + '</td>' +
        '</tr>';
    }).join('') || '<tr><td colspan="12" class="empty">尚无逐车记录。</td></tr>';

    const rejected = (state.activity.rejected_receipts || []).map(function (item) {
      return '<tr><td>' + FL.escape(item.vehicle_id) + '</td><td>v' + item.version + '</td><td>' +
        FL.escape(FL.rejectReasonText(item.reason)) + '</td><td>' + FL.escape(item.detail) + '</td><td>' +
        FL.escape(item.reported_at) + '</td></tr>';
    }).join('') || '<tr><td colspan="5" class="empty">没有被拒回执。</td></tr>';

    const timeouts = (state.activity.timeout_events || []).map(function (item) {
      return '<tr><td>' + FL.escape(item.vehicle_id) + '</td><td>v' + item.version + '</td><td>' +
        FL.escape(item.timed_out_at) + '</td><td>' + FL.escape(item.resolved_at || '未解决') + '</td></tr>';
    }).join('') || '<tr><td colspan="4" class="empty">没有超时事件。</td></tr>';

    const audits = (state.activity.audits || []).map(function (item) {
      return '<tr><td>' + FL.escape(item.created_at) + '</td><td>' + FL.escape(item.actor_id) + '</td><td>' +
        FL.escape(item.action) + '</td><td>' + FL.escape(item.detail) + '</td></tr>';
    }).join('') || '<tr><td colspan="4" class="empty">暂无审计。</td></tr>';

    detail.innerHTML =
      '<div class="row"><div class="actions-row" style="margin:0">' + actionButtons + '</div></div>' +
      (actions.promote_blocked_reason ? '<div class="blocked">推广不可用：' + FL.escape(actions.promote_blocked_reason) + '</div>' : '') +
      '<div class="row"><strong>状态</strong>' + FL.badge(activity.status) + '<span class="muted">' + meta.join(' · ') + '</span></div>' +
      (state.activity.application ? '<div class="row"><strong>申请原因</strong><span>' + FL.escape(state.activity.application.reason) + '</span></div>' : '') +
      scopeHtml +
      '<div class="summary-grid">' +
        summaryItem('总车辆', summary.total) +
        summaryItem('已发', summary.released) +
        summaryItem('未发', summary.not_released) +
        summaryItem('成功', summary.success) +
        summaryItem('失败', summary.failed) +
        summaryItem('超时', summary.timeout) +
        summaryItem('未确认', summary.waiting) +
        summaryItem('生效数', summary.effective) +
        summaryItem('被拒回执', summary.rejected_receipts) +
      '</div>' +
      '<h3>逐车核查</h3>' +
      '<div class="table-wrap"><table><thead><tr><th>车辆</th><th>批次</th><th>版本</th><th>基线</th><th>期望</th><th>生效</th><th>状态</th><th>失败原因</th><th>下发时间</th><th>回执时间</th><th>超时/迟到</th><th>重复</th></tr></thead><tbody>' + rows + '</tbody></table></div>' +
      '<h3>被拒回执</h3>' +
      '<div class="table-wrap"><table><thead><tr><th>车辆</th><th>版本</th><th>原因</th><th>说明</th><th>时间</th></tr></thead><tbody>' + rejected + '</tbody></table></div>' +
      '<h3>超时事件历史</h3>' +
      '<div class="table-wrap"><table><thead><tr><th>车辆</th><th>版本</th><th>超时时间</th><th>解决时间</th></tr></thead><tbody>' + timeouts + '</tbody></table></div>' +
      '<h3>活动审计</h3>' +
      '<div class="table-wrap"><table><thead><tr><th>时间</th><th>操作人</th><th>动作</th><th>详情</th></tr></thead><tbody>' + audits + '</tbody></table></div>';

    detail.querySelectorAll('[data-action]').forEach(function (button) {
      button.addEventListener('click', function () { guard(function () { return runAction(button.dataset.action); }); });
    });
  }

  function summaryItem(label, value) {
    return '<div class="summary-item"><span>' + label + '</span><b>' + (value === undefined || value === null ? 0 : value) + '</b></div>';
  }

  async function runAction(action) {
    const activity = state.activity && state.activity.activity;
    if (!activity) return;
    const id = activity.id;
    let path = '';
    let body;
    if (action === 'start_pilot') path = '/activities/' + id + '/pilot';
    if (action === 'promote') path = '/activities/' + id + '/promote';
    if (action === 'approve_restore') path = '/activities/' + id + '/restore/approve';
    if (action === 'start_restore') path = '/activities/' + id + '/restore/start';
    if (action === 'complete') path = '/activities/' + id + '/complete';
    if (action === 'restored') path = '/activities/' + id + '/restored';
    if (action === 'stop') {
      const reason = window.prompt('停止原因（可留空）：', '试点失败，暂停推广');
      if (reason === null) return;
      path = '/activities/' + id + '/stop';
      body = { reason: reason };
    }
    state.activity = await FL.api(path, { method: 'POST', body: body });
    FL.toast('操作成功', 'ok');
    renderActivity();
    await loadHistory();
    await loadApplications();
  }

  async function loadHistory() {
    const data = await FL.api('/activities');
    const body = document.querySelector('#activityHistoryTable tbody');
    const items = (data && data.items) || [];
    if (items.length === 0) {
      body.innerHTML = '<tr><td colspan="5" class="empty">暂无活动。</td></tr>';
      return;
    }
    body.innerHTML = items.map(function (item) {
      return '<tr><td>' + FL.escape(item.created_at) + '</td><td>' + item.interval_seconds + ' 秒</td><td>' +
        (item.pilot_vehicles || []).join('、') + '</td><td>' + FL.badge(item.status) + '</td><td>' +
        '<button class="btn small" data-open="' + FL.escape(item.id) + '">查看</button></td></tr>';
    }).join('');
    body.querySelectorAll('[data-open]').forEach(function (button) {
      button.addEventListener('click', function () { guard(function () { return openActivity(button.dataset.open); }); });
    });
  }

  async function loadVehicles() {
    const data = await FL.api('/vehicles');
    state.vehicles = (data && data.items) || [];
    const body = document.querySelector('#vehicleTable tbody');
    const canToggle = FL.profile && FL.profile.role === 'O';
    body.innerHTML = state.vehicles.map(function (vehicle) {
      const toggle = canToggle
        ? '<button class="btn small" data-toggle="' + FL.escape(vehicle.id) + '" data-online="' + (vehicle.online ? '1' : '0') + '">' +
          (vehicle.online ? '设为离线' : '设为在线') + '</button>'
        : '<span class="muted">仅运营可切换</span>';
      return '<tr><td><strong>' + FL.escape(vehicle.id) + '</strong></td>' +
        '<td>' + (vehicle.online ? '<span class="badge green">在线</span>' : '<span class="badge gray">离线</span>') + '</td>' +
        '<td>v' + vehicle.current_version + '</td>' +
        '<td>' + vehicle.expected_interval + ' 秒</td>' +
        '<td>v' + vehicle.effective_version + '</td>' +
        '<td>' + vehicle.effective_interval + ' 秒</td>' +
        '<td>' + FL.badge(vehicle.latest_release_status || 'not_released') + '</td>' +
        '<td>' + toggle + '</td></tr>';
    }).join('') || '<tr><td colspan="8" class="empty">车辆数据为空。</td></tr>';
    body.querySelectorAll('[data-toggle]').forEach(function (button) {
      button.addEventListener('click', function () {
        guard(function () { return toggleVehicle(button.dataset.toggle, button.dataset.online !== '1'); });
      });
    });
    renderCreatePanel();
  }

  async function toggleVehicle(id, online) {
    await FL.api('/vehicles/' + id + '/online', { method: 'POST', body: { online: online } });
    FL.toast(id + ' 已切换为' + (online ? '在线' : '离线'), 'ok');
    await loadVehicles();
  }

  async function loadAudits() {
    const data = await FL.api('/audits?limit=200');
    const body = document.querySelector('#auditTable tbody');
    const items = (data && data.items) || [];
    body.innerHTML = items.map(function (item) {
      return '<tr><td>' + FL.escape(item.created_at) + '</td><td>' + FL.escape(item.actor_id) + '</td><td>' +
        FL.escape(item.actor_role || '-') + '</td><td>' + FL.escape(item.action) + '</td><td>' +
        FL.escape(item.target_type) + ':' + FL.escape(item.target_id) + '</td><td>' + FL.escape(item.detail) + '</td></tr>';
    }).join('') || '<tr><td colspan="6" class="empty">暂无审计记录。</td></tr>';
  }
})();
