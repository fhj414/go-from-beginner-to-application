(() => {
  const LS_KEY = "gq_client_state_v2";
  const DEMO_ID_KEY = "gq_demo_resume_id";
  const DEMO_NAME_KEY = "gq_demo_resume_name";

  const $ = (id) => document.getElementById(id);

  const state = {
    user: null,
    curriculum: null,
    activeLesson: null,
    pickIdx: null,
    orderPerm: [],
    orderSel: null,
    wxJssdk: false,
  };

  const stages = [
    { id: "seed", icon: "🥚", name: "种子选手", desc: "刚出发也很棒" },
    { id: "sprout", icon: "🌱", name: "发芽新手", desc: "开始懂 Go 了" },
    { id: "scout", icon: "🐹", name: "见习探险家", desc: "能自己闯几关啦" },
    { id: "ninja", icon: "🥷", name: "并发小忍者", desc: "核心知识在成形" },
    { id: "star", icon: "🎓", name: "星球小明星", desc: "主线毕业纪念" },
  ];

  function api(path, opts = {}) {
    return fetch(path, {
      credentials: "include",
      headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
      ...opts,
    }).then(async (res) => {
      const text = await res.text();
      let data;
      try {
        data = text ? JSON.parse(text) : {};
      } catch {
        data = { raw: text };
      }
      if (!res.ok) {
        const err = new Error(data.error || res.statusText || "request failed");
        err.status = res.status;
        err.data = data;
        throw err;
      }
      return data;
    });
  }

  function isWeChat() {
    return /micromessenger/i.test(navigator.userAgent || "");
  }

  function loadScript(src) {
    return new Promise((resolve, reject) => {
      const key = encodeURIComponent(src);
      if (document.querySelector(`script[data-src="${key}"]`)) {
        resolve();
        return;
      }
      const s = document.createElement("script");
      s.src = src;
      s.dataset.src = key;
      s.onload = resolve;
      s.onerror = () => reject(new Error(`load ${src}`));
      document.head.appendChild(s);
    });
  }

  function stageMeta(stage) {
    return stages.find((item) => item.id === stage) || stages[0];
  }

  function fmtSecs(n) {
    const s = Math.max(0, Math.floor(Number(n) || 0));
    if (s < 60) return `${s} 秒`;
    const m = Math.floor(s / 60);
    const r = s % 60;
    return `${m} 分 ${r} 秒`;
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function showScreen(name) {
    ["screen-landing", "screen-map", "screen-play"].forEach((id) => {
      $(id).classList.toggle("hidden", id !== `screen-${name}`);
    });
  }

  function shareURL() {
    return `${window.location.origin}/static/index.html`;
  }

  function saveLocal() {
    if (!state.user) return;
    const snap = {
      current_lesson: state.user.progress.current_lesson,
      completed: state.user.progress.completed,
      xp: state.user.progress.xp,
      total_study_secs: state.user.total_study_secs,
    };
    localStorage.setItem(LS_KEY, JSON.stringify(snap));
    if (state.user.source === "demo") {
      localStorage.setItem(DEMO_ID_KEY, state.user.id);
      localStorage.setItem(DEMO_NAME_KEY, state.user.nickname || "");
    }
  }

  function loadLocal() {
    try {
      const raw = localStorage.getItem(LS_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  }

  function mergeLocalUser(serverUser) {
    const loc = loadLocal();
    if (!loc || !serverUser) return serverUser;
    if ((loc.xp || 0) > (serverUser.progress?.xp || 0)) {
      serverUser.progress.xp = loc.xp;
    }
    const srvDone = Object.keys(serverUser.progress?.completed || {}).filter(
      (k) => serverUser.progress.completed[k],
    ).length;
    const locDone = Object.keys(loc.completed || {}).filter((k) => loc.completed[k]).length;
    if (locDone > srvDone) {
      serverUser.progress.completed = { ...loc.completed };
      serverUser.progress.current_lesson = Math.max(
        serverUser.progress.current_lesson,
        loc.current_lesson || 1,
      );
    }
    if ((loc.total_study_secs || 0) > (serverUser.total_study_secs || 0)) {
      serverUser.total_study_secs = loc.total_study_secs;
    }
    return serverUser;
  }

  function savedDemo() {
    const id = localStorage.getItem(DEMO_ID_KEY) || "";
    const name = localStorage.getItem(DEMO_NAME_KEY) || "探险小地鼠";
    return id.startsWith("demo:") ? { id, name } : null;
  }

  function renderLanding() {
    const resume = savedDemo();
    if (resume && !$("nick").value.trim()) {
      $("nick").value = resume.name;
    }
    $("btnResumeDemo").classList.toggle("hidden", !resume);
    $("resumeHint").textContent = resume
      ? `已记住演示身份「${resume.name}」，随时可以继续上次的冒险。`
      : "演示模式也会记住你的闯关进度；配置微信后可切换为真实授权登录。";
  }

  function setHeaderFromUser() {
    if (!state.user) {
      $("mascot").textContent = "🥚";
      $("stageLine").textContent = "加载中…";
      return;
    }
    const meta = stageMeta(state.user.progress.last_stage || "seed");
    $("mascot").textContent = meta.icon;
    $("stageLine").textContent = `${state.user.nickname} · ${meta.name}`;
  }

  function progressSummary() {
    const p = state.user?.progress;
    if (!p) return { done: 0, total: 0, current: 1 };
    return {
      done: p.completed_count || 0,
      total: p.total_lessons || state.curriculum?.lessons?.length || 0,
      current: p.current_lesson || 1,
    };
  }

  function renderBadgeStrip() {
    const strip = $("badgeStrip");
    strip.innerHTML = "";
    const current = state.user?.progress?.last_stage || "seed";
    const currentIdx = stages.findIndex((item) => item.id === current);
    stages.forEach((item, idx) => {
      const div = document.createElement("button");
      div.type = "button";
      div.className = "badge-item";
      if (idx <= currentIdx) div.classList.add("unlocked");
      if (item.id === current) div.classList.add("active");
      div.innerHTML = `<span class="badge-icon">${item.icon}</span><strong>${item.name}</strong><small>${item.desc}</small>`;
      div.addEventListener("click", () => {
        $("reminder").textContent = `${item.icon} ${item.name}：${item.desc}`;
      });
      strip.append(div);
    });
  }

  function renderMap() {
    const u = state.user;
    const c = state.curriculum;
    if (!u || !c) return;
    const meta = stageMeta(u.progress.last_stage || "seed");
    const summary = progressSummary();
    const percent = Math.max(0, Math.min(100, u.progress_percent || 0));

    $("playerBadge").textContent = meta.icon;
    $("stageChip").textContent = meta.name;
    $("playerName").textContent = u.nickname;
    $("playerDesc").textContent = `${meta.desc}，今天再闯一关就更熟啦。`;
    $("reminder").textContent = u.progress.reminder_note || "挑一关开始吧，小地鼠陪你～";
    $("xpVal").textContent = u.progress.xp;
    $("studyVal").textContent = fmtSecs(u.total_study_secs);
    $("doneVal").textContent = `${summary.done}/${summary.total}`;
    $("goalVal").textContent =
      summary.current > summary.total ? "邀请朋友来挑战" : `第 ${summary.current} 关`;
    $("progressLine").textContent = `已完成 ${summary.done}/${summary.total} 关`;
    $("nextLine").textContent =
      summary.current > summary.total
        ? "主线通关啦，可以刷榜和分享"
        : `下一站：第 ${summary.current} 关 · ${u.progress.resume_title || "继续冒险"}`;
    $("progressFill").style.width = `${percent}%`;

    const path = $("pathList");
    path.innerHTML = "";
    c.lessons.forEach((lesson) => {
      const done = !!u.progress.completed[lesson.id];
      const unlocked = lesson.id <= summary.current || done;
      const row = document.createElement("li");
      const btn = document.createElement("button");
      btn.type = "button";
      btn.disabled = !unlocked;
      if (done) btn.classList.add("done");
      if (lesson.id === summary.current && !done) btn.classList.add("focus");
      const status = done ? "已掌握" : unlocked ? "可挑战" : "未解锁";
      btn.innerHTML = `
        <span class="badge">${done ? "✅" : unlocked ? "🎮" : "🔒"}</span>
        <div class="meta">
          <strong>第 ${lesson.id} 关 · ${escapeHtml(lesson.title)}</strong>
          <span>${escapeHtml(lesson.subtitle)} · ${status} · +${lesson.reward_xp} XP</span>
        </div>
      `;
      btn.addEventListener("click", () => startLesson(lesson.id));
      row.append(btn);
      path.append(row);
    });

    renderBadgeStrip();
    setHeaderFromUser();
  }

  function shufflePerm(n) {
    const arr = Array.from({ length: n }, (_, i) => i);
    for (let i = n - 1; i > 0; i -= 1) {
      const j = Math.floor(Math.random() * (i + 1));
      [arr[i], arr[j]] = [arr[j], arr[i]];
    }
    if (n > 1 && arr.every((v, i) => v === i)) return shufflePerm(n);
    return arr;
  }

  function permMatchesAnswer(perm, ans) {
    return !!perm && !!ans && perm.length === ans.length && perm.every((v, i) => v === ans[i]);
  }

  function startLesson(id) {
    const lesson = state.curriculum.lessons.find((item) => item.id === id);
    if (!lesson) return;
    state.activeLesson = lesson;
    state.pickIdx = null;
    state.orderSel = null;
    state.orderPerm = lesson.kind === "order" ? shufflePerm(lesson.order_items.length) : [];

    $("lessonTag").textContent = `${lesson.subtitle} · 第 ${lesson.id} 关`;
    $("lessonReward").textContent = `+${lesson.reward_xp} XP`;
    $("lessonTitle").textContent = lesson.title;
    $("lessonStory").textContent = lesson.story;
    $("coachLine").textContent =
      lesson.id <= 3
        ? "地鼠教练：不用急，先理解感觉就好。"
        : lesson.id <= 7
          ? "地鼠教练：你已经进入 Go 核心区啦。"
          : "地鼠教练：这几关开始碰到简单应用了。";
    $("lessonQ").textContent = lesson.question;
    $("lessonHint").textContent = `提示：${lesson.hint}`;
    $("feedback").className = "toast hidden";

    const body = $("lessonBody");
    body.innerHTML = "";
    if (lesson.kind === "pick_one") {
      const wrap = document.createElement("div");
      wrap.className = "options";
      lesson.options.forEach((text, idx) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "opt";
        btn.textContent = text;
        btn.addEventListener("click", () => {
          state.pickIdx = idx;
          wrap.querySelectorAll(".opt").forEach((el) => el.classList.remove("selected"));
          btn.classList.add("selected");
        });
        wrap.append(btn);
      });
      body.append(wrap);
    } else {
      const tip = document.createElement("p");
      tip.className = "fineprint";
      tip.textContent = "点一个步骤，再点另一个，就能交换位置。";
      body.append(tip);
      const row = document.createElement("div");
      row.className = "order-row";
      row.id = "orderRow";
      body.append(row);
      renderOrderRow();
    }
    showScreen("play");
  }

  function renderOrderRow() {
    const lesson = state.activeLesson;
    const row = $("orderRow");
    if (!lesson || !row) return;
    row.innerHTML = "";
    state.orderPerm.forEach((itemIdx, pos) => {
      const item = document.createElement("div");
      item.className = "order-item";
      if (state.orderSel === pos) item.classList.add("selected");
      item.dataset.pos = String(pos);
      item.innerHTML = `<span class="idx">${pos + 1}</span><span>${lesson.order_items[itemIdx]}</span>`;
      item.addEventListener("click", () => onOrderTap(pos));
      row.append(item);
    });
  }

  function onOrderTap(pos) {
    if (state.orderSel === null) {
      state.orderSel = pos;
      renderOrderRow();
      return;
    }
    if (state.orderSel === pos) {
      state.orderSel = null;
      renderOrderRow();
      return;
    }
    const a = state.orderSel;
    const b = pos;
    [state.orderPerm[a], state.orderPerm[b]] = [state.orderPerm[b], state.orderPerm[a]];
    state.orderSel = null;
    renderOrderRow();
  }

  function lessonCorrect() {
    const lesson = state.activeLesson;
    if (!lesson) return false;
    if (lesson.kind === "pick_one") {
      if (state.pickIdx === null) return false;
      return lesson.any_choice ? true : state.pickIdx === lesson.correct;
    }
    return permMatchesAnswer(state.orderPerm, lesson.order_answer);
  }

  async function submitProgress(patch) {
    const u = state.user;
    const body = {
      current_lesson: patch.current_lesson ?? u.progress.current_lesson,
      completed: { ...u.progress.completed, ...(patch.completed || {}) },
      xp: patch.xp ?? u.progress.xp,
    };
    const res = await api("/api/progress", {
      method: "PUT",
      body: JSON.stringify(body),
    });
    state.user = mergeLocalUser(res.user);
    saveLocal();
  }

  async function onCheck() {
    const fb = $("feedback");
    const lesson = state.activeLesson;
    if (!lesson) return;
    fb.className = "toast";
    if (lesson.kind === "pick_one" && state.pickIdx === null) {
      fb.textContent = "先选一个答案吧，小地鼠在等你点一下。";
      fb.classList.add("bad");
      return;
    }
    if (!lessonCorrect()) {
      fb.textContent = "差一点点，再看看提示，马上就会啦。";
      fb.classList.add("bad");
      return;
    }
    const u = state.user;
    const total = state.curriculum.lessons.length || 1;
    const already = !!u.progress.completed[lesson.id];
    const nextLesson = Math.min(total + 1, Math.max(u.progress.current_lesson, lesson.id + 1));
    await submitProgress({
      completed: { [lesson.id]: true },
      xp: already ? u.progress.xp : u.progress.xp + (lesson.reward_xp || 10),
      current_lesson: lesson.id >= u.progress.current_lesson ? nextLesson : u.progress.current_lesson,
    });
    const meta = stageMeta(state.user.progress.last_stage || "seed");
    fb.textContent = already
      ? `这关你已经很熟啦，继续冲刺下一关。`
      : `闯关成功！+${lesson.reward_xp} XP，当前身份：${meta.icon} ${meta.name}`;
    fb.classList.add("ok");
    setHeaderFromUser();
    setTimeout(() => {
      showScreen("map");
      renderMap();
    }, 1100);
  }

  async function refreshMe() {
    const res = await api("/api/me");
    state.user = mergeLocalUser(res.user);
    saveLocal();
    renderMap();
  }

  async function demoLogin(resumeID = "") {
    const nickname = ($("nick").value || localStorage.getItem(DEMO_NAME_KEY) || "探险小地鼠").trim();
    const res = await api("/api/auth/demo", {
      method: "POST",
      body: JSON.stringify({ nickname, resume_id: resumeID }),
    });
    state.user = mergeLocalUser(res.user);
    $("nick").value = state.user.nickname || nickname;
    saveLocal();
    showScreen("map");
    renderMap();
    initWxJssdkShare().catch(() => {});
  }

  async function boot() {
    try {
      state.curriculum = await api("/api/curriculum");
      try {
        const me = await api("/api/me");
        state.user = mergeLocalUser(me.user);
      } catch (err) {
        if (err.status !== 401) throw err;
      }
      if (state.user) {
        saveLocal();
        showScreen("map");
        renderMap();
      } else {
        renderLanding();
        showScreen("landing");
      }
    } catch (err) {
      console.error(err);
      $("stageLine").textContent = "网络开小差了，刷新试试";
    }
  }

  async function loadWxMeta() {
    try {
      const meta = await api("/api/wechat/enabled");
      state.wxJssdk = !!meta.jssdk;
      $("btnWx").classList.toggle("hidden", !meta.enabled);
      $("wxShareHint").classList.toggle("hidden", !(isWeChat() && state.wxJssdk));
    } catch {
      $("btnWx").classList.add("hidden");
    }
  }

  async function initWxJssdkShare() {
    if (!isWeChat() || !state.wxJssdk) return;
    const pageURL = location.href.split("#")[0];
    let cfg;
    try {
      const res = await fetch(`/api/wechat/jssdk-config?url=${encodeURIComponent(pageURL)}`, {
        credentials: "include",
      });
      if (!res.ok) return;
      cfg = await res.json();
    } catch {
      return;
    }
    try {
      await loadScript("https://res.wx.qq.com/open/js/jweixin-1.6.0.js");
    } catch {
      return;
    }
    const wx = window.wx;
    if (!wx || !wx.config) return;
    wx.config({
      debug: !!cfg.debug,
      appId: cfg.app_id,
      timestamp: cfg.timestamp,
      nonceStr: cfg.nonce_str,
      signature: cfg.signature,
      jsApiList: cfg.js_api_list || ["updateAppMessageShareData", "updateTimelineShareData"],
    });
    wx.ready(() => {
      const shareData = {
        title: cfg.share_title,
        desc: cfg.share_desc,
        link: cfg.share_link,
        imgUrl: cfg.share_img,
      };
      wx.updateAppMessageShareData(shareData);
      wx.updateTimelineShareData({
        title: cfg.share_title,
        link: cfg.share_link,
        imgUrl: cfg.share_img,
      });
    });
  }

  async function openLeaderboard() {
    const { items } = await api("/api/leaderboard?limit=30");
    const list = $("lbList");
    list.innerHTML = "";
    if (!items.length) {
      const li = document.createElement("li");
      li.textContent = "还没有排行数据，先闯一会儿吧。";
      list.append(li);
    } else {
      items.forEach((row) => {
        const me = state.user && row.id === state.user.id;
        const meta = stageMeta(row.stage || "seed");
        const li = document.createElement("li");
        if (me) li.classList.add("me");
        li.innerHTML = `
          <span class="rk">${row.rank}</span>
          <span class="lb-name">${meta.icon} ${escapeHtml(row.nickname)}</span>
          <span class="lb-side">${fmtSecs(row.total_study_secs)} · ${row.xp} XP</span>
        `;
        list.append(li);
      });
    }
    $("dlgLb").showModal();
  }

  function shareText() {
    const name = state.user?.nickname ? `${state.user.nickname} 邀请你` : "邀请你";
    return `${name}一起玩《小地鼠闯 Go 星球》，边玩边学 Go 基础。`;
  }

  async function nativeShare() {
    const payload = {
      title: "小地鼠闯 Go 星球",
      text: shareText(),
      url: shareURL(),
    };
    if (navigator.share) {
      await navigator.share(payload);
      return;
    }
    await navigator.clipboard.writeText(`${payload.text} ${payload.url}`);
    alert("已复制分享文案和链接。");
  }

  function openShare() {
    $("shareCopy").textContent = shareText();
    $("shareUrl").textContent = shareURL();
    $("dlgShare").showModal();
  }

  $("btnLb").addEventListener("click", () => openLeaderboard().catch(console.error));
  $("btnShare").addEventListener("click", openShare);
  $("btnQuickShare").addEventListener("click", openShare);
  $("btnNativeShare").addEventListener("click", () => nativeShare().catch(console.error));
  $("btnCopy").addEventListener("click", async () => {
    const text = `${shareText()} ${shareURL()}`;
    try {
      await navigator.clipboard.writeText(text);
      alert("已复制，可以发给朋友啦。");
    } catch {
      prompt("复制下面内容", text);
    }
  });
  $("btnCheck").addEventListener("click", () => onCheck().catch(console.error));
  $("btnBack").addEventListener("click", () => {
    showScreen("map");
    renderMap();
  });
  $("btnContinue").addEventListener("click", () => {
    if (!state.user) return;
    const id = Math.min(
      state.user.progress.current_lesson || 1,
      state.curriculum?.lessons?.length || 1,
    );
    startLesson(id);
  });
  $("btnDemo").addEventListener("click", () => demoLogin().catch(console.error));
  $("btnResumeDemo").addEventListener("click", () => {
    const resume = savedDemo();
    if (!resume) return;
    $("nick").value = resume.name || "";
    demoLogin(resume.id).catch(console.error);
  });
  $("btnWx").addEventListener("click", () => {
    window.location.href = "/api/auth/wechat/start";
  });
  $("btnLogout").addEventListener("click", async () => {
    await api("/api/auth/logout", { method: "POST" });
    state.user = null;
    renderLanding();
    showScreen("landing");
  });
  document.querySelectorAll("[data-close]").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.getElementById(btn.getAttribute("data-close")).close();
    });
  });

  setInterval(() => {
    if (document.visibilityState !== "visible" || !state.user) return;
    api("/api/study/tick", {
      method: "POST",
      body: JSON.stringify({ seconds: 30 }),
    })
      .then((res) => {
        state.user.total_study_secs = res.total_study_secs;
        $("studyVal").textContent = fmtSecs(res.total_study_secs);
        saveLocal();
      })
      .catch(() => {});
  }, 30000);

  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && state.user) {
      refreshMe().catch(() => {});
    }
  });

  loadWxMeta();
  boot()
    .then(() => initWxJssdkShare())
    .catch(() => {});
})();
