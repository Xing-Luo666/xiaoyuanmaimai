/**
 * 公共站点头部组件
 * 在子页面调用 SiteHeader.render({ activeCat: 'electronics' }) 注入完整顶部导航
 * 需先引入 api.js
 */
(function() {
  function escHtml(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // 分类定义（与主页保持一致）
  var CATS = [
    { key: 'all',           label: '全部宝贝',     icon: '',     href: 'products.html' },
    { key: 'electronics',   label: '数码电子',     icon: '📱 ',  href: 'products.html?category=electronics' },
    { key: 'books',         label: '书籍教材',     icon: '📖 ',  href: 'products.html?category=books' },
    { key: 'furniture',     label: '生活家居',     icon: '🏠 ',  href: 'products.html?category=furniture' },
    { key: 'sports',        label: '运动户外',     icon: '⚽ ',  href: 'products.html?category=sports' },
    { key: 'entertainment', label: '娱乐玩具',     icon: '🎮 ',  href: 'products.html?category=entertainment' },
    { key: 'clothing',      label: '服饰鞋包',     icon: '👗 ',  href: 'products.html?category=clothing' },
    { key: 'beauty',        label: '美妆护肤',     icon: '💄 ',  href: 'products.html?category=beauty' }
  ];

  function buildHTML(opts) {
    opts = opts || {};
    var activeCat = opts.activeCat || '';
    var cats = CATS.map(function(c) {
      var cls = (c.key === activeCat) ? 'cat-nav-item active' : 'cat-nav-item';
      return '<div class="' + cls + '" onclick="location.href=\'' + c.href + '\'">' + c.icon + c.label + '</div>';
    }).join('');

    return ''
      + '<div class="site-header-wrap">'
      +   '<div class="top-nav">'
      +     '<div class="top-nav-inner">'
      +       '<span>欢迎来到校园二手交易平台！</span>'
      +       '<a id="dbStatusBar" href="db-config.html" style="display:none;align-items:center;gap:6px;font-size:12px;text-decoration:none;color:inherit;">'
      +         '<span style="color:var(--text-light)">数据库:</span>'
      +         '<span id="dbStatusDot" style="display:inline-block;width:8px;height:8px;border-radius:50%;background:#9CA3AF"></span>'
      +         '<span id="dbStatusText" style="color:var(--text-light)">检测中...</span>'
      +       '</a>'
      +       '<a id="adminLink" href="admin.html" style="display:none;font-size:12px;color:var(--text-light);text-decoration:none;">数据管理</a>'
      +       '<div id="topNavRight">'
      +         '<a href="login.html">请登录</a>'
      +         '<a href="register.html">免费注册</a>'
      +       '</div>'
      +     '</div>'
      +   '</div>'
      +   '<header class="main-header subpage">'
      +     '<div class="header-inner">'
      +       '<a href="javascript:void(0)" class="site-back-btn" onclick="if(history.length>1)history.back();else location.href=\'../index.html\'" title="返回"><span class="arrow">←</span></a>'
      +       '<div class="logo-area" onclick="location.href=\'../index.html\'" style="cursor:pointer">'
      +         '<div class="logo-icon">📚</div>'
      +         '<div>'
      +           '<div class="logo-text">校园二手</div>'
      +           '<div class="logo-sub">闲置交易平台</div>'
      +         '</div>'
      +       '</div>'
      +       '<div class="search-area">'
      +         '<input type="text" id="mainSearch" placeholder="搜索你想要的闲置好物..." onkeydown="if(event.key===\'Enter\')SiteHeader.doSearch()">'
      +         '<button class="search-btn" onclick="SiteHeader.doSearch()">搜索</button>'
      +       '</div>'
      +       '<div class="header-icons">'
      +         '<div class="header-icon-item" onclick="location.href=\'chat-list.html\'" title="消息" style="position:relative">'
      +           '💬 <span>消息</span>'
      +           '<span id="chatBadge" style="display:none;position:absolute;top:-4px;right:2px;min-width:18px;height:18px;line-height:18px;text-align:center;background:#ef4444;color:#fff;border-radius:9px;font-size:10px;padding:0 4px"></span>'
      +         '</div>'
      +         '<div class="header-icon-item" onclick="location.href=\'orders.html\'" title="我的订单">'
      +           '📋 <span>订单</span>'
      +         '</div>'
      +         '<div class="header-icon-item" onclick="location.href=\'publish.html\'" title="发布闲置">'
      +           '➕ <span>发布</span>'
      +         '</div>'
      +         '<div class="header-icon-item" onclick="location.href=\'profile.html\'" title="个人中心">'
      +           '<div style="width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#3B82F6,#2563EB);color:#fff;display:flex;align-items:center;justify-content:center;font-size:13px;font-weight:700" id="headerAvatar">👤</div>'
      +           '<span>我的</span>'
      +         '</div>'
      +       '</div>'
      +     '</div>'
      +   '</header>'
      +   '<nav class="cat-nav">'
      +     '<div class="cat-nav-inner">' + cats + '</div>'
      +   '</nav>'
      + '</div>';
  }

  function updateUserUI() {
    var user = api.getUser();
    if (!user) return;
    var topNavRight = document.getElementById('topNavRight');
    if (topNavRight) {
      topNavRight.innerHTML =
        '<span>Hi, <b>' + escHtml(user.username) + '</b></span>' +
        '<a href="orders.html">我的订单</a>' +
        '<a href="publish.html">我要卖闲置</a>' +
        '<a href="profile.html">个人中心</a>' +
        '<a href="javascript:void(0)" onclick="api.logout();location.href=\'../index.html\'">退出</a>';
    }
    var avatarEl = document.getElementById('headerAvatar');
    if (avatarEl && user.username) {
      avatarEl.textContent = user.username[0].toUpperCase();
    }
    checkDbStatusForAdmin();
  }

  async function checkDbStatusForAdmin() {
    var user = api.getUser();
    if (!user || user.role !== 'admin') return;
    var bar = document.getElementById('dbStatusBar');
    var dot = document.getElementById('dbStatusDot');
    var text = document.getElementById('dbStatusText');
    var adminLink = document.getElementById('adminLink');
    if (!bar) return;
    bar.style.display = 'flex';
    adminLink.style.display = 'inline';
    var status = await api.checkDbStatus();
    if (status && status.connected) {
      dot.style.background = '#22C55E';
      text.textContent = '已连接';
      text.style.color = '#22C55E';
    } else {
      dot.style.background = '#EF4444';
      text.textContent = '未连接';
      text.style.color = '#EF4444';
    }
  }

  function updateChatBadge() {
    var badge = document.getElementById('chatBadge');
    if (!badge) return;
    if (!api.token) { badge.style.display = 'none'; return; }
    api.get('/chat-unread').then(function(res) {
      if (res.code === 200 && res.data && res.data.count > 0) {
        badge.style.display = 'inline-block';
        badge.textContent = res.data.count > 99 ? '99+' : res.data.count;
      } else {
        badge.style.display = 'none';
      }
    }).catch(function() { if (badge) badge.style.display = 'none'; });
  }

  function doSearch() {
    var kw = document.getElementById('mainSearch').value.trim();
    location.href = 'products.html' + (kw ? ('?keyword=' + encodeURIComponent(kw)) : '');
  }

  // 从URL推断当前分类
  function inferActiveCat() {
    var params = new URLSearchParams(location.search);
    var cat = params.get('category') || 'all';
    return cat;
  }

  window.SiteHeader = {
    render: function(opts) {
      opts = opts || {};
      if (!opts.activeCat) opts.activeCat = inferActiveCat();
      // 找到页面中已有的header占位元素替换，否则插入到 body 最前面
      var placeholder = document.getElementById('siteHeaderPlaceholder');
      var html = buildHTML(opts);
      if (placeholder) {
        placeholder.outerHTML = html;
      } else {
        document.body.insertAdjacentHTML('afterbegin', html);
      }
      // 初始化用户状态
      api.verifyAndInit().then(function(ok) { if (ok) updateUserUI(); });
      updateChatBadge();
      setInterval(updateChatBadge, 15000);
    },
    doSearch: doSearch,
    updateUserUI: updateUserUI
  };
})();
