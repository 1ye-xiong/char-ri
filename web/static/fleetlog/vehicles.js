(function () {
  const FL = window.FL;
  const VEHICLES = [
    { id: 'V1', token: 'v1-token' },
    { id: 'V2', token: 'v2-token' },
    { id: 'V3', token: 'v3-token' },
    { id: 'V4', token: 'v4-token' },
    { id: 'V5', token: 'v5-token' }
  ];
  const state = { versions: {} };

  document.addEventListener('DOMContentLoaded', init);

  function init() {
    renderCards();
    document.getElementById('refreshVehicleStates').addEventListener('click', loadStates);
    if (FL.profile) {
      document.getElementById('vehicleTimeoutInfo').textContent =
        '超时判定 ' + (FL.profile.timeoutSeconds || 60) + ' 秒 · 当前身份 ' + FL.roleName(FL.profile.role);
    }
    loadStates();
  }

  function renderCards() {
    const container = document.getElementById('vehicleCards');
    container.innerHTML = VEHICLES.map(function (vehicle) {
      return '' +
        '<article class="vehicle-card" data-vehicle="' + vehicle.id + '">' +
          '<h3>' + vehicle.id + ' <span class="badge gray" data-online>状态未知</span></h3>' +
          '<label>车辆令牌<input data-token value="' + vehicle.token + '"></label>' +
          '<label>回执车辆编号<input data-claim value="' + vehicle.id + '"></label>' +
          '<label>上报版本<input data-version type="number" min="1" value="1"></label>' +
          '<label>失败原因<select data-reason>' +
            '<option value="应用失败">应用失败</option>' +
            '<option value="配置解析失败">配置解析失败</option>' +
            '<option value="版本写入失败">版本写入失败</option>' +
          '</select></label>' +
          '<label>最近拉取结果<input data-last readonly value="尚未拉取"></label>' +
          '<div class="vehicle-actions">' +
            '<button class="btn small primary" data-do="pull">上线拉取</button>' +
            '<button class="btn small" data-do="success">上报成功</button>' +
            '<button class="btn small danger" data-do="failure">上报失败</button>' +
            '<button class="btn small ghost" data-do="duplicate">重复成功</button>' +
            '<button class="btn small ghost" data-do="stale">旧版本上报</button>' +
            '<button class="btn small ghost" data-do="cross">跨车回执</button>' +
            '<button class="btn small ghost hidden" data-do="toggle">切换在线</button>' +
          '</div>' +
          '<div class="vehicle-result" data-result></div>' +
        '</article>';
    }).join('');

    VEHICLES.forEach(function (vehicle) {
      const card = container.querySelector('[data-vehicle="' + vehicle.id + '"]');
      card.querySelectorAll('[data-do]').forEach(function (button) {
        button.addEventListener('click', function () {
          handleAction(vehicle, card, button.dataset.do).catch(function (err) {
            setResult(card, '请求失败：' + err.message, false);
          });
        });
      });
    });
  }

  async function loadStates() {
    try {
      const data = await FL.api('/vehicles');
      const items = (data && data.items) || [];
      items.forEach(function (item) {
        const card = document.querySelector('[data-vehicle="' + item.id + '"]');
        if (!card) return;
        const badge = card.querySelector('[data-online]');
        badge.className = 'badge ' + (item.online ? 'green' : 'gray');
        badge.textContent = item.online ? '在线' : '离线';
        card.querySelector('[data-last]').value =
          '期望 v' + item.current_version + ' / ' + item.expected_interval + ' 秒 · 生效 v' +
          item.effective_version + ' / ' + item.effective_interval + ' 秒 · ' + FL.statusText(item.latest_release_status || 'not_released');
        state.versions[item.id] = item.current_version;
        const toggle = card.querySelector('[data-do="toggle"]');
        if (FL.profile && FL.profile.role === 'O') {
          toggle.classList.remove('hidden');
          toggle.textContent = item.online ? '设为离线' : '设为在线';
          toggle.dataset.online = item.online ? '1' : '0';
        }
      });
    } catch (err) {
      if (err.status !== 401) {
        FL.toast('车辆状态加载失败：' + err.message, 'err');
      }
    }
  }

  async function handleAction(vehicle, card, action) {
    const token = card.querySelector('[data-token]').value.trim();
    const claim = card.querySelector('[data-claim]').value.trim() || vehicle.id;
    const versionInput = card.querySelector('[data-version]');
    const reason = card.querySelector('[data-reason]').value;

    if (action === 'toggle') {
      const online = card.querySelector('[data-do="toggle"]').dataset.online !== '1';
      await FL.api('/vehicles/' + vehicle.id + '/online', { method: 'POST', body: { online: online } });
      setResult(card, vehicle.id + ' 已切换为' + (online ? '在线' : '离线'), true);
      await loadStates();
      return;
    }

    if (action === 'pull') {
      const data = await FL.api('/vehicle/pull', {
        method: 'POST',
        vehicleToken: token,
        body: { vehicle_id: claim }
      });
      versionInput.value = data.version;
      state.versions[vehicle.id] = data.version;
      setResult(card, '拉取成功：v' + data.version + ' / ' + data.interval_seconds + ' 秒（' +
        (data.has_release ? '发布状态 ' + FL.statusText(data.release_status) : '初始配置，无发布记录') + '）', true);
      return;
    }

    let version = Number(versionInput.value) || 1;
    let result = 'success';
    let failureReason = '';
    let claimed = claim;

    if (action === 'failure') {
      result = 'failure';
      failureReason = reason;
    }
    if (action === 'stale') {
      version = Math.max(1, version - 1);
      versionInput.value = version;
    }
    if (action === 'cross') {
      claimed = vehicle.id === 'V2' ? 'V1' : 'V2';
      card.querySelector('[data-claim]').value = claimed;
    }

    const data = await FL.api('/vehicle/receipts', {
      method: 'POST',
      vehicleToken: token,
      body: {
        vehicle_id: claimed,
        version: version,
        result: result,
        failure_reason: failureReason
      }
    });
    const prefix = data.accepted ? '已接受' : '被拒绝';
    const detail = data.accepted
      ? (data.message + '，发布状态 ' + FL.statusText(data.release_status) + '，生效 ' + data.effective_interval + ' 秒')
      : (FL.rejectReasonText(data.reason) + '：' + data.message);
    setResult(card, prefix + '：' + detail, data.accepted);
    await loadStates();
  }

  function setResult(card, message, ok) {
    const box = card.querySelector('[data-result]');
    box.textContent = message;
    box.className = 'vehicle-result ' + (ok ? 'ok' : 'fail');
  }
})();
