(function () {
  const TOKEN_KEY = 'fleetlog_token';
  const PROFILE_KEY = 'fleetlog_profile';

  function readProfile() {
    try {
      return JSON.parse(localStorage.getItem(PROFILE_KEY) || 'null');
    } catch (err) {
      return null;
    }
  }

  const FL = {
    token: localStorage.getItem(TOKEN_KEY) || '',
    profile: readProfile(),

    saveSession(data) {
      this.token = data.token;
      this.profile = {
        userId: data.user_id,
        role: data.role,
        displayName: data.display_name,
        timeoutSeconds: data.timeout_seconds
      };
      localStorage.setItem(TOKEN_KEY, this.token);
      localStorage.setItem(PROFILE_KEY, JSON.stringify(this.profile));
    },

    clearSession() {
      this.token = '';
      this.profile = null;
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(PROFILE_KEY);
    },

    async api(path, options) {
      options = options || {};
      const headers = {};
      if (options.body !== undefined) {
        headers['Content-Type'] = 'application/json';
      }
      if (this.token) {
        headers['Authorization'] = 'Bearer ' + this.token;
      }
      if (options.vehicleToken) {
        headers['X-Vehicle-Token'] = options.vehicleToken;
      }
      const response = await fetch('/api/v1/fleetlog' + path, {
        method: options.method || 'GET',
        headers: headers,
        body: options.body !== undefined ? JSON.stringify(options.body) : undefined
      });
      let payload = null;
      try {
        payload = await response.json();
      } catch (err) {
        payload = null;
      }
      if (!response.ok) {
        const body = (payload && payload.error) || {};
        const error = new Error(body.message || ('HTTP ' + response.status));
        error.code = body.code || '';
        error.status = response.status;
        throw error;
      }
      return payload ? payload.data : null;
    },

    roleName(role) {
      return { R: '研发申请人', A: '审批负责人', O: '发布运营' }[role] || role || '';
    },

    statusText(status) {
      return {
        draft: '草稿', pending: '待审批', rejected: '已退回', approved: '已批准', voided: '已作废',
        pilot_pending: '待试点', piloting: '试点中', pilot_passed: '试点通过', promoting: '推广中',
        completed: '已完成', stopped: '已停止', restoring: '恢复中', restored: '恢复完成',
        waiting: '等待确认', success: '成功', failed: '失败', timeout: '超时', not_released: '未发布'
      }[status] || status || '';
    },

    statusClass(status) {
      return {
        draft: 'gray', pending: 'orange', rejected: 'red', approved: 'blue', voided: 'gray',
        pilot_pending: 'blue', piloting: 'orange', pilot_passed: 'blue', promoting: 'orange',
        completed: 'green', stopped: 'red', restoring: 'orange', restored: 'green',
        waiting: 'gray', success: 'green', failed: 'red', timeout: 'orange', not_released: 'gray'
      }[status] || 'gray';
    },

    batchText(batch) {
      return { pilot: '试点', promotion: '推广', restore: '恢复' }[batch] || '未发布';
    },

    rejectReasonText(reason) {
      return {
        cross_vehicle: '跨车回执', unknown_version: '未知版本', stale_version: '过期回执',
        duplicate_receipt: '重复回执', invalid_payload: '内容不完整'
      }[reason] || reason;
    },

    badge(status) {
      return '<span class="badge ' + this.statusClass(status) + '">' + this.statusText(status) + '</span>';
    },

    escape(value) {
      return String(value === undefined || value === null ? '' : value)
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    },

    toast(message, kind) {
      const box = document.getElementById('toast');
      if (!box) return;
      box.textContent = message;
      box.className = 'toast ' + (kind || '');
      clearTimeout(this._toastTimer);
      this._toastTimer = setTimeout(function () { box.className = 'toast hidden'; }, 4200);
    }
  };

  window.FL = FL;
})();
