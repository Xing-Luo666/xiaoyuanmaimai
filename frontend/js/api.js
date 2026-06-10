// API 基础路径：同域部署时使用相对路径，跨域部署时可通过 window.API_BASE 覆盖
const API_BASE = window.API_BASE || '/api';

const SSO_KEY = 'sso_token';
const USER_KEY = 'sso_user';
const EXPIRES_KEY = 'sso_expires';

const api = {
  _token: sessionStorage.getItem(SSO_KEY) || localStorage.getItem(SSO_KEY) || '',
  _ready: false,
  _initPromise: null,

  get token() {
    return this._token || sessionStorage.getItem(SSO_KEY) || localStorage.getItem(SSO_KEY) || '';
  },

  set token(t) {
    this._token = t;
    if (t) {
      sessionStorage.setItem(SSO_KEY, t);
      localStorage.setItem(SSO_KEY, t);
    } else {
      sessionStorage.removeItem(SSO_KEY);
      localStorage.removeItem(SSO_KEY);
      sessionStorage.removeItem(USER_KEY);
      localStorage.removeItem(USER_KEY);
      sessionStorage.removeItem(EXPIRES_KEY);
      localStorage.removeItem(EXPIRES_KEY);
    }
  },

  setToken(t) {
    this.token = t;
  },

  isLoggedIn() {
    const t = this.token;
    if (!t) return false;
    const expiresAt = sessionStorage.getItem(EXPIRES_KEY) || localStorage.getItem(EXPIRES_KEY);
    if (expiresAt) {
      const now = Math.floor(Date.now() / 1000);
      if (parseInt(expiresAt) <= now) {
        this.logoutQuiet();
        return false;
      }
    }
    return true;
  },

  async verifyAndInit() {
    if (this._ready) return this.token ? true : false;
    if (this._initPromise) return this._initPromise;
    
    this._initPromise = (async () => {
      const t = this.token;
      if (!t) {
        this._ready = true;
        return false;
      }
      try {
        const rawRes = await fetch(API_BASE + '/verify-token?token=' + encodeURIComponent(t));
        const json = await rawRes.json();
        if (json.code === 200) {
          this._ready = true;
          return true;
        } else {
          this.logoutQuiet();
          this._ready = true;
          return false;
        }
      } catch(e) {
        this._ready = true;
        return false;
      }
    })();
    
    return this._initPromise;
  },

  async request(method, path, data, skipAuth) {
    const opts = {
      method,
      headers: { 'Content-Type': 'application/json' },
    };
    const t = this.token;
    if (t && !skipAuth) {
      opts.headers['Authorization'] = 'Bearer ' + t;
    }
    if (data && method !== 'GET') {
      opts.body = JSON.stringify(data);
    }
    const params = method === 'GET' && data ? '?' + new URLSearchParams(data) : '';
    const url = API_BASE + path + params;

    let res;
    try {
      res = await fetch(url, opts);
    } catch(e) {
      return { code: -1, message: '网络错误，请确认后端已启动' };
    }

    let json;
    try {
      json = await res.json();
    } catch(e) {
      return { code: res.status, message: '服务器返回异常' };
    }

    if (json.code === 401) {
      this.logoutQuiet();
      const currentPath = window.location.pathname;
      if (!currentPath.includes('login.html') && !currentPath.includes('register.html')) {
        const redirect = encodeURIComponent(window.location.href);
        window.location.href = window.location.origin + '/pages/login.html?redirect=' + redirect;
      }
    }

    return json;
  },

  get(path, params, skipAuth) { return this.request('GET', path, params, skipAuth); },
  post(path, data) { return this.request('POST', path, data); },
  put(path, data) { return this.request('PUT', path, data); },
  del(path) { return this.request('DELETE', path); },

  async login(username, password) {
    const res = await this.post('/login', { username, password });
    if (res.code === 200) {
      this.token = res.data.token;
      const userInfo = JSON.stringify({
        userId: res.data.userId,
        username: res.data.username,
        role: res.data.role,
        expiresAt: res.data.expiresAt
      });
      sessionStorage.setItem(USER_KEY, userInfo);
      localStorage.setItem(USER_KEY, userInfo);
      sessionStorage.setItem(EXPIRES_KEY, String(res.data.expiresAt));
      localStorage.setItem(EXPIRES_KEY, String(res.data.expiresAt));
      this._ready = true;
    }
    return res;
  },

  async register(data) {
    const res = await this.post('/register', data);
    if (res.code === 200) {
      this.token = res.data.token;
      const userInfo = JSON.stringify({
        userId: res.data.userId,
        username: res.data.username,
        role: res.data.role,
        expiresAt: res.data.expiresAt
      });
      sessionStorage.setItem(USER_KEY, userInfo);
      localStorage.setItem(USER_KEY, userInfo);
      sessionStorage.setItem(EXPIRES_KEY, String(res.data.expiresAt));
      localStorage.setItem(EXPIRES_KEY, String(res.data.expiresAt));
      this._ready = true;
    }
    return res;
  },

  logout() {
    this.token = '';
    localStorage.removeItem(SSO_KEY);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(EXPIRES_KEY);
    sessionStorage.removeItem(SSO_KEY);
    sessionStorage.removeItem(USER_KEY);
    sessionStorage.removeItem(EXPIRES_KEY);
    window.location.href = window.location.origin + '/pages/login.html';
  },

  logoutQuiet() {
    this._token = '';
    localStorage.removeItem(SSO_KEY);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(EXPIRES_KEY);
    sessionStorage.removeItem(SSO_KEY);
    sessionStorage.removeItem(USER_KEY);
    sessionStorage.removeItem(EXPIRES_KEY);
    this._ready = false;
  },

  getUser() {
    const u = sessionStorage.getItem(USER_KEY) || localStorage.getItem(USER_KEY);
    return u ? JSON.parse(u) : null;
  },

  async checkDbStatus() {
    try {
      const res = await this.get('/db-config');
      return res.code === 200 ? res.data : null;
    } catch(e) {
      return null;
    }
  },

  // 购物车
  async getCart() { return this.get('/cart'); },
  async addToCart(data) { return this.post('/cart', data); },
  async updateCartItem(id, data) { return this.put('/cart/' + id, data); },
  async removeCartItems(ids) {
    if (Array.isArray(ids) && ids.length) {
      return this.del('/cart?ids=' + ids.join(','));
    }
    return this.del('/cart/' + ids);
  },

  // 收藏
  async getFavorites() { return this.get('/favorites'); },
  async toggleFavorite(data) { return this.post('/favorites', data); },
  async checkFavorites(ids) {
    return this.get('/favorites/check?ids=' + ids.join(','));
  },

  // 点赞
  async toggleLike(productId) { return this.post('/products/' + productId + '/like'); },

  // 历史
  async getHistory() { return this.get('/history'); },
  async addHistory(data) { return this.post('/history', data); },

  // 聊天
  async getChatHistory(orderId) { return this.get('/chat/' + orderId); }
};

(function exposeGlobals() {
  window.SSO = api;
})();

function showToast(message, type) {
  const icons = { success: '\u2713', error: '\u2715', warning: '!' };
  const el = document.createElement('div');
  el.className = 'mall-toast ' + (type || 'success');
  el.textContent = message;
  document.body.appendChild(el);
  setTimeout(function() { el.style.opacity = '0'; el.style.transition = 'opacity .3s'; setTimeout(function() { el.remove(); }, 300); }, 2500);
}

function formatPrice(price) {
  return '\u00A5' + Number(price).toFixed(2);
}

function formatDate(dateStr) {
  if (!dateStr) return '';
  var d = new Date(dateStr);
  if (isNaN(d.getTime())) return dateStr;
  var now = new Date();
  var diff = now - d;
  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前';
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前';
  if (diff < 604800000) return Math.floor(diff / 86400000) + '天前';
  return d.toLocaleDateString('zh-CN');
}

var CATEGORIES = [
  { value: 'electronics', label: '\u6570\u7801\u7535\u5B50' },
  { value: 'books', label: '\u4E66\u7C4D\u6559\u6750' },
  { value: 'clothing', label: '\u670D\u9970\u978B\u5305' },
  { value: 'furniture', label: '\u751F\u6D3B\u5BB6\u5C45' },
  { value: 'sports', label: '\u8FD0\u52A8\u5668\u6750' },
  { value: 'entertainment', label: '\u5A31\u4E50\u73A9\u5177' },
  { value: 'beauty', label: '\u7F8E\u5986\u62A4\u80A4' },
  { value: 'other', label: '\u5176\u4ED6' }
];

var CAMPUSES = [
  { value: 'hangkong', label: '航空港校区' },
  { value: 'longquanyi', label: '龙泉驿校区' },
  { value: 'main', label: '主校区' },
  { value: 'south', label: '南校区' },
  { value: 'north', label: '北校区' },
  { value: 'east', label: '东校区' },
  { value: 'west', label: '西校区' }
];

var BUILDINGS = Array.from({length:21}, function(_,i){ return {value: String(i+1), label: (i+1)+'栋'}; });
BUILDINGS.push({value: 'other', label: '其他'});

var CONDITIONS = [
  { value: 'new', label: '\u5168\u65B0' },
  { value: 'like_new', label: '\u51E0\u4E4E\u5168\u65B0' },
  { value: 'good', label: '\u826F\u597D' },
  { value: 'fair', label: '\u4E00\u822C' },
  { value: 'old', label: '\u8001\u65E7' }
];

function getCategoryLabel(v) {
  var c = CATEGORIES.find(function(x) { return x.value === v; });
  return c ? c.label : v;
}

function getCampusLabel(v) {
  var c = CAMPUSES.find(function(x) { return x.value === v; });
  return c ? c.label : v;
}

function getConditionLabel(v) {
  var c = CONDITIONS.find(function(x) { return x.value === v; });
  return c ? c.label : v;
}

function getStatusLabel(status) {
  var map = {
    selling: { text: '\u5728\u552E', cls: 'el-tag--success' },
    reserved: { text: '\u5DF2\u9884\u8BA2', cls: 'el-tag--warning' },
    sold: { text: '\u5DF2\u552E\u51FA', cls: 'el-tag--danger' },
    pending: { text: '\u5F85\u786E\u8BA4', cls: 'el-tag--warning' },
    accepted: { text: '\u5DF2\u63A5\u53D7', cls: 'el-tag--success' },
    rejected: { text: '\u5DF2\u62D2\u7EDD', cls: 'el-tag--danger' },
    completed: { text: '\u5DF2\u5B8C\u6210', cls: 'el-tag--info' },
    cancelled: { text: '\u5DF2\u53D6\u6D88', cls: 'el-tag--danger' }
  };
  return map[status] || { text: status, cls: 'el-tag--info' };
}
