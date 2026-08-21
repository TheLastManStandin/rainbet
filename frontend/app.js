const gridSizes = [25, 36, 49, 64];
const storageKey = "rainbet-mines-auth";
const soundKey = "rainbet-mines-sound";
const gridSizeKey = "rainbet-mines-grid-size";
const bombFrameCount = 28;
const explosionDuration = 700;
const houseEdge = 4;

const sparkleSlots = {
  1: { top: "25%", left: "40%", scale: 0.8, size: "50%" },
  2: { top: "-14%", left: "3%", scale: 0.9, size: "55%" },
  3: { top: "15%", left: "-12%", scale: 0.9, size: "30%" },
  4: { top: "75%", left: "37%", scale: 1.1, size: "25%" },
  5: { top: "12%", left: "73%", scale: 1.1, size: "40%" },
};

const state = {
  auth: loadAuth(),
  balance: null,
  mode: "manual",
  gridSize: loadGridSize(),
  mines: 8,
  betAmount: "10.00",
  game: null,
  cells: [],
  selected: new Set(),
  autoRunning: false,
  loading: false,
  soundOn: localStorage.getItem(soundKey) !== "off",
};

const elements = {
  accountName: document.querySelector("#account-name"),
  authError: document.querySelector("#auth-error"),
  authForm: document.querySelector("#auth-form"),
  authOverlay: document.querySelector("#auth-overlay"),
  authPassword: document.querySelector("#auth-password"),
  authUsername: document.querySelector("#auth-username"),
  autoField: document.querySelector("#auto-field"),
  autoRounds: document.querySelector("#auto-rounds"),
  balanceValue: document.querySelector("#balance-value"),
  betAmount: document.querySelector("#bet-amount"),
  betDouble: document.querySelector("#bet-double"),
  betHalf: document.querySelector("#bet-half"),
  betMode: document.querySelector("#bet-mode"),
  board: document.querySelector("#mines-board"),
  bombCount: document.querySelector("#bomb-count"),
  changeUser: document.querySelector("#change-user"),
  gameContent: document.querySelector("#game-content"),
  gemCount: document.querySelector("#gem-count"),
  gridOptions: document.querySelector("#grid-options"),
  inputError: document.querySelector("#input-error"),
  mineCount: document.querySelector("#mine-count"),
  minesTopBar: document.querySelector("#mines-top-bar"),
  minesThumb: document.querySelector("#mines-thumb"),
  minesTrack: document.querySelector(".bar-content-container"),
  minesBar: document.querySelector(".bar-content"),
  multiplierCurrent: document.querySelector("#multiplier-current"),
  multiplierNext: document.querySelector("#multiplier-next"),
  placeBet: document.querySelector("#place-bet"),
  placeBetLabel: document.querySelector("#place-bet-label"),
  profitContainer: document.querySelector("#profit-container"),
  profitCurrent: document.querySelector("#profit-current"),
  profitNext: document.querySelector("#profit-next"),
  soundToggle: document.querySelector("#sound-toggle"),
  tileNotice: document.querySelector("#tile-notice"),
  toast: document.querySelector("#toast"),
  topUp: document.querySelector("#top-up"),
  winAmount: document.querySelector("#win-amount"),
  winDisplay: document.querySelector("#win-display"),
  winMultiplier: document.querySelector("#win-multiplier"),
};

const bombFrames = [];
const sounds = {
  click: new Audio("/assets/sounds/button-click.mp3"),
  gem: new Audio("/assets/sounds/final-win-4.mp3"),
  bomb: new Audio("/assets/sounds/bomb_5.mp3"),
  win: new Audio("/assets/sounds/win-display-4.mp3"),
};

let toastTimer;
let noticeTimer;

function preloadAssets() {
  for (let index = 1; index <= bombFrameCount; index += 1) {
    const frame = new Image();
    frame.src = `/assets/mine/${index}.webp`;
    bombFrames.push(frame);
  }
  Object.values(sounds).forEach((sound) => {
    sound.preload = "auto";
    sound.volume = 0.5;
  });
}

function playSound(name) {
  if (!state.soundOn) return;
  const source = sounds[name];
  if (!source) return;
  const clip = source.cloneNode();
  clip.volume = source.volume;
  void clip.play().catch(() => {});
}

function loadAuth() {
  try {
    return JSON.parse(localStorage.getItem(storageKey)) || null;
  } catch {
    return null;
  }
}

function saveAuth(auth) {
  state.auth = auth;
  localStorage.setItem(storageKey, JSON.stringify(auth));
}

function loadGridSize() {
  const stored = Number(localStorage.getItem(gridSizeKey));
  return gridSizes.includes(stored) ? stored : 25;
}

function getDiamonds() {
  return state.gridSize - state.mines;
}

function multiplierFor(gridSize, mines, picks) {
  const diamonds = gridSize - mines;
  if (picks <= 0 || picks > diamonds) return null;

  let chance = 1;
  for (let step = 0; step < picks; step += 1) {
    chance = (chance * (diamonds - step)) / (gridSize - step);
  }
  if (chance === 0) return null;

  const value = (100 - houseEdge) / (chance * 100);
  return Math.min(Math.floor(value * 100) / 100, 1000000);
}

function formatMultiplier(value) {
  return value === null ? "—" : `${value.toFixed(2)}x`;
}

function formatMoney(value) {
  return `$${Number(value || 0).toFixed(2)}`;
}

function setError(message = "") {
  elements.inputError.textContent = message;
}

function showToast(message, isError = false) {
  clearTimeout(toastTimer);
  elements.toast.textContent = message;
  elements.toast.classList.toggle("error", isError);
  elements.toast.classList.add("show");
  toastTimer = setTimeout(() => elements.toast.classList.remove("show"), 3200);
}

function showNotice(message) {
  clearTimeout(noticeTimer);
  elements.tileNotice.querySelector("span").textContent = message;
  elements.tileNotice.classList.add("is-active");
  noticeTimer = setTimeout(() => elements.tileNotice.classList.remove("is-active"), 2600);
}

function showAuth(message = "") {
  elements.authOverlay.classList.remove("is-hidden");
  elements.authError.textContent = message;
  elements.authUsername.focus();
}

function hideAuth() {
  elements.authOverlay.classList.add("is-hidden");
  elements.authError.textContent = "";
}

function createTile(index) {
  const tile = document.createElement("button");
  tile.className = "grid-item";
  tile.type = "button";
  tile.setAttribute("role", "gridcell");
  tile.setAttribute("aria-label", `Плитка ${index + 1}`);
  tile.style.touchAction = "manipulation";

  const reveal = document.createElement("div");
  reveal.className = "reveal-transition-item";

  const icon = document.createElement("div");
  icon.className = "grid-icon";

  const gem = document.createElement("div");
  gem.className = "grid-icon-gem";

  const gemInner = document.createElement("div");
  gemInner.className = "grid-icon-gem-inner";

  const shine = document.createElement("div");
  shine.className = "line-shine";
  shine.style.maskImage = "url(/assets/diamond-final.png)";
  shine.style.webkitMaskImage = "url(/assets/diamond-final.png)";

  const shineInner = document.createElement("div");
  shineInner.className = "line-shine-inner";
  shine.append(shineInner);

  const gemImage = document.createElement("img");
  gemImage.src = "/assets/diamond-final.png";
  gemImage.alt = "";
  gemImage.loading = "eager";

  gemInner.append(shine, gemImage);
  gem.append(gemInner);
  icon.append(gem);

  const overlay = document.createElement("div");
  overlay.className = "grid-item-overlay-hover";

  tile.append(reveal, icon, overlay);

  tile.addEventListener("mouseenter", () => {
    if (isTilePlayable(index)) tile.classList.add("hovered");
  });
  tile.addEventListener("mouseleave", () => tile.classList.remove("hovered"));
  tile.addEventListener("click", () => handleTileClick(index));

  return { tile, icon, gem, gemInner, gemImage };
}

function buildBoard() {
  elements.board.className = `mines-grid mines-grid-${state.gridSize}`;
  elements.board.replaceChildren();
  state.cells = [];

  for (let index = 0; index < state.gridSize; index += 1) {
    const cell = createTile(index);
    state.cells.push({ ...cell, revealed: false });
    elements.board.append(cell.tile);
  }

  applySelection();
}

function applySelection() {
  state.cells.forEach((cell, index) => {
    cell.tile.classList.toggle("active", state.mode === "auto" && state.selected.has(index));
  });
}

function isTilePlayable(index) {
  const cell = state.cells[index];
  if (!cell || cell.revealed || state.loading) return false;
  if (state.mode === "auto") return !state.autoRunning;
  return Boolean(state.game && state.game.status === "inProcess");
}

function addSparkles(cell) {
  const slots = [1, 2, 3, 4, 5].sort(() => 0.5 - Math.random()).slice(0, 3);
  const animations = [1, 2, 3, 4, 5]
    .map((index) => `sparkleAnimation${index}`)
    .sort(() => 0.5 - Math.random())
    .slice(0, 3);

  slots.forEach((slot, position) => {
    const config = sparkleSlots[slot];
    const container = document.createElement("div");
    container.className = "sparkle-container";
    container.style.position = "absolute";
    container.style.top = config.top;
    container.style.left = config.left;
    container.style.width = config.size;
    container.style.height = config.size;
    container.style.transform = `scale(${config.scale})`;
    container.style.zIndex = "1";
    container.style.animationName = animations[position];
    container.style.animationDuration = `${(Math.random() * 1.6 + 3.9).toFixed(2)}s`;
    container.style.animationDelay = `${(Math.random() * 0.3 + 0.1).toFixed(2)}s`;
    container.style.animationIterationCount = "infinite";
    container.style.animationTimingFunction = "ease-in-out";

    const image = document.createElement("img");
    image.src = slot % 2 === 0 ? "/assets/sparkle-2.webp" : "/assets/sparkle-1.webp";
    image.alt = "";
    container.append(image);
    cell.gemInner.insertBefore(container, cell.gemImage);
  });
}

function addMultiplierBadge(tile, multiplier) {
  if (multiplier === null) return;
  const wrapper = document.createElement("div");
  wrapper.className = "grid-item-multiplier";
  const badge = document.createElement("div");
  badge.textContent = `${multiplier.toFixed(2)}x`;
  wrapper.append(badge);
  tile.append(wrapper);
}

function buildBombIcon() {
  const bomb = document.createElement("div");
  bomb.className = "grid-icon-bomb";
  const image = document.createElement("img");
  image.src = "/assets/bomb.svg";
  image.alt = "";
  bomb.append(image);
  return bomb;
}

function revealGem(index, multiplier, picked = true) {
  const cell = state.cells[index];
  if (!cell || cell.revealed) return;

  cell.revealed = true;
  cell.tile.classList.remove("hovered", "active");
  cell.tile.classList.add("grid-item-revealed");
  cell.tile.classList.add(picked ? "grid-item-gem" : "grid-item-none");
  cell.tile.classList.remove("grid-item");

  void cell.tile.offsetWidth;
  cell.gem.classList.add(picked ? "grid-item-picked" : "not-picked");
  if (picked) addSparkles(cell);

  if (picked) addMultiplierBadge(cell.tile, multiplier);
}

function revealMine(index, isFinal) {
  const cell = state.cells[index];
  if (!cell || cell.revealed) return;

  cell.revealed = true;
  cell.tile.classList.remove("hovered", "active", "grid-item");
  cell.tile.classList.add("grid-item-revealed", isFinal ? "grid-item-mine" : "grid-item-none");
  cell.icon.replaceChildren();

  if (!isFinal) {
    cell.icon.append(buildBombIcon());
    return;
  }

  const container = document.createElement("div");
  container.className = "bomb-animation-container";

  const still = document.createElement("div");
  still.className = "bomb-still-image";
  const stillInner = document.createElement("div");
  stillInner.className = "bomb-still-image-inner";
  const stillImage = document.createElement("img");
  stillImage.src = "/assets/bomb.svg";
  stillImage.alt = "";
  stillInner.append(stillImage);
  still.append(stillInner);
  container.append(still);
  cell.icon.append(container);

  playExplosion(container, still);
}

function revealOtherMine(index) {
  const cell = state.cells[index];
  if (!cell || cell.revealed) return;

  cell.revealed = true;
  cell.tile.classList.remove("hovered", "active", "grid-item");
  cell.tile.classList.add("grid-item-revealed", "grid-item-none");
  cell.icon.replaceChildren(buildBombIcon());
}

function playExplosion(container, still) {
  const width = container.clientWidth;
  const height = container.clientHeight;
  const first = bombFrames[0];
  if (!width || !height || !first || !first.naturalWidth) {
    still.classList.add("active");
    return;
  }

  const canvas = document.createElement("canvas");
  canvas.className = "bomb-animation-canvas";
  canvas.width = width * 1.5;
  canvas.height = height * 1.5;
  container.insertBefore(canvas, still);

  const context = canvas.getContext("2d");
  const scaleX = canvas.width / first.naturalWidth;
  const scaleY = canvas.height / first.naturalHeight;
  const startedAt = performance.now();
  const frameInterval = explosionDuration / (bombFrameCount - 1);

  const timer = setInterval(() => {
    const progress = Math.min((performance.now() - startedAt) / explosionDuration, 1);
    const frame = bombFrames[Math.min(Math.floor(progress * (bombFrameCount - 1)), bombFrameCount - 1)];

    context.clearRect(0, 0, canvas.width, canvas.height);
    if (frame && frame.naturalWidth) {
      context.save();
      context.scale(scaleX, scaleY);
      context.drawImage(frame, 0, 0);
      context.restore();
    }

    if (progress < 1) return;

    clearInterval(timer);
    canvas.remove();
    still.classList.add("active");
  }, frameInterval);
}

function revealRemaining(mineIndexes, finalIndex) {
  const mines = new Set(mineIndexes || []);
  state.cells.forEach((cell, index) => {
    if (cell.revealed || index === finalIndex) return;
    if (mines.has(index)) revealOtherMine(index);
    else revealGem(index, null, false);
  });
}

function resetBoard() {
  elements.winDisplay.classList.add("is-hidden");
  buildBoard();
}

function renderAccount() {
  const username = state.auth && state.auth.username;
  elements.accountName.textContent = username || "Гость";
  elements.balanceValue.textContent = state.balance === null ? "—" : formatMoney(state.balance);
  elements.changeUser.textContent = username ? "Выйти" : "Войти";
}

function renderSlider() {
  const maxGems = state.gridSize - 1;
  elements.mineCount.max = String(maxGems);
  if (state.mines > maxGems) state.mines = maxGems;
  elements.mineCount.value = String(getDiamonds());

  elements.gemCount.textContent = String(getDiamonds());
  elements.bombCount.textContent = String(state.mines);

  const fraction = (getDiamonds() - 1) / (maxGems - 1);
  elements.minesTopBar.style.left = `calc((100% + 18px) * ${fraction} - 9px)`;
  elements.minesThumb.style.left = `${fraction * 100}%`;
}

function renderProfit() {
  const inGame = Boolean(state.game && state.game.status === "inProcess");
  elements.profitContainer.classList.toggle("is-hidden", !inGame);
  if (!inGame) return;

  const opened = state.game.openedCells.length;
  const bet = Number(state.game.betAmount) || 0;
  const current = opened > 0 ? multiplierFor(state.gridSize, state.mines, opened) : 0;
  const next = multiplierFor(state.gridSize, state.mines, opened + 1);

  elements.multiplierCurrent.textContent = opened > 0 ? formatMultiplier(current) : "0.00x";
  elements.multiplierNext.textContent = formatMultiplier(next);
  elements.profitCurrent.textContent = formatMoney(opened > 0 ? bet * current : 0);
  elements.profitNext.textContent = formatMoney(next === null ? 0 : bet * next);
}

function renderControls() {
  const inGame = Boolean(state.game && state.game.status === "inProcess");
  const locked = inGame || state.loading || state.autoRunning;

  elements.betAmount.value = state.betAmount;
  elements.betAmount.disabled = locked;
  elements.betHalf.disabled = locked;
  elements.betDouble.disabled = locked;
  elements.mineCount.disabled = locked;
  elements.autoRounds.disabled = locked;
  elements.topUp.disabled = state.loading;
  elements.changeUser.disabled = locked;

  [...elements.gridOptions.children].forEach((option) => {
    const active = Number(option.dataset.gridSize) === state.gridSize;
    option.classList.toggle("is-active", active);
    option.setAttribute("aria-checked", String(active));
    option.disabled = locked;
  });

  [...elements.betMode.children].forEach((tab) => {
    const active = tab.dataset.mode === state.mode;
    tab.classList.toggle("is-active", active);
    tab.setAttribute("aria-selected", String(active));
    tab.disabled = locked;
  });

  elements.autoField.classList.toggle("is-hidden", state.mode !== "auto");

  elements.placeBet.classList.remove("is-cashout", "is-stop");
  if (!state.auth) {
    elements.placeBetLabel.textContent = "Войти";
    elements.placeBet.disabled = state.loading;
  } else if (state.mode === "auto") {
    elements.placeBetLabel.textContent = state.autoRunning ? "Остановить автоигру" : "Начать автоигру";
    elements.placeBet.classList.toggle("is-stop", state.autoRunning);
    elements.placeBet.disabled = state.loading && !state.autoRunning;
  } else if (inGame) {
    const opened = state.game.openedCells.length;
    const multiplier = opened > 0 ? multiplierFor(state.gridSize, state.mines, opened) : null;
    elements.placeBetLabel.textContent =
      opened > 0 ? `Забрать ${formatMoney(Number(state.game.betAmount) * multiplier)}` : "Откройте плитку";
    elements.placeBet.classList.add("is-cashout");
    elements.placeBet.disabled = opened === 0 || state.loading;
  } else {
    elements.placeBetLabel.textContent = "Ставка";
    elements.placeBet.disabled = state.loading;
  }

  renderSlider();
  renderProfit();
}

function setLoading(loading) {
  state.loading = loading;
  renderControls();
}

function newClientSeed() {
  if (window.crypto && window.crypto.randomUUID) {
    return `web-${window.crypto.randomUUID()}`;
  }
  return `web-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  headers.set("Content-Type", "application/json");

  if (state.auth && state.auth.username && state.auth.password) {
    headers.set("Authorization", `Basic ${btoa(`${state.auth.username}:${state.auth.password}`)}`);
  }

  const response = await fetch(`/api${path}`, { ...options, headers, credentials: "omit" });
  const body = await response.json().catch(() => ({}));

  if (response.status === 401) {
    showAuth("Сессия недействительна, войдите заново.");
    throw new Error("Authentication required");
  }
  if (!response.ok) {
    throw new Error(body.error || "Запрос не удалось выполнить.");
  }

  return body;
}

async function refreshBalance() {
  const auth = state.auth;
  if (!auth) {
    state.balance = null;
    renderAccount();
    return;
  }

  try {
    const account = await api("/user");
    if (state.auth === auth) state.balance = account.balance;
  } catch (error) {
    if (state.auth === auth) state.balance = null;
    if (error.message !== "Authentication required") showToast(error.message, true);
  } finally {
    renderAccount();
  }
}

function readBet() {
  const amount = elements.betAmount.value.trim().replace(",", ".");
  if (!amount || !Number.isFinite(Number(amount)) || Number(amount) <= 0) {
    setError("Введите сумму ставки больше $0.00.");
    return null;
  }
  return Number(amount).toFixed(2);
}

async function startGame() {
  setError();
  if (state.game && state.game.status === "inProcess") return;

  const amount = readBet();
  if (amount === null) return;

  state.betAmount = amount;
  resetBoard();
  setLoading(true);

  try {
    const game = await api("/mines/bets", {
      method: "POST",
      body: JSON.stringify({
        betAmount: Number(amount),
        gridSize: state.gridSize,
        mines: state.mines,
        demo: false,
        clientSeed: newClientSeed(),
      }),
    });

    state.game = {
      id: game.id,
      status: game.status,
      betAmount: amount,
      openedCells: [],
      multiplier: null,
    };
    void refreshBalance();
    return true;
  } catch (error) {
    if (error.message !== "Authentication required") setError(error.message);
    return false;
  } finally {
    setLoading(false);
  }
}

async function openCell(cellIndex) {
  if (!state.game || state.game.status !== "inProcess") return null;

  setLoading(true);
  try {
    const move = await api(`/mines/bets/${state.game.id}/moves`, {
      method: "POST",
      body: JSON.stringify({ cellIndex }),
    });

    state.game.status = move.status;
    state.game.openedCells = move.openedCells;
    state.game.multiplier = move.multiplier || null;

    if (move.result === "bomb") {
      playSound("bomb");
      revealMine(cellIndex, true);
      revealRemaining(move.mineIndexes, cellIndex);
      state.game.status = "failed";
    } else {
      playSound("gem");
      revealGem(cellIndex, multiplierFor(state.gridSize, state.mines, move.openedCells.length));
    }

    return move;
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
    return null;
  } finally {
    setLoading(false);
  }
}

async function cashOut() {
  if (!state.game || state.game.status !== "inProcess" || !state.game.openedCells.length) return null;

  setLoading(true);
  try {
    const cashout = await api(`/mines/bets/${state.game.id}/cashout`, { method: "POST", body: "{}" });
    state.game.status = cashout.status;
    state.game.payout = cashout.payout;
    state.game.multiplier = cashout.multiplier;

    revealRemaining(cashout.mineIndexes, -1);
    playSound("win");

    elements.winMultiplier.textContent = `${cashout.multiplier}x`;
    elements.winAmount.textContent = formatMoney(cashout.payout);
    elements.winDisplay.classList.remove("is-hidden");

    void refreshBalance();
    return cashout;
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
    return null;
  } finally {
    setLoading(false);
  }
}

async function topUpBalance() {
  if (!state.auth) {
    showAuth();
    return;
  }

  setLoading(true);
  try {
    await api("/user/balance", {
      method: "POST",
      body: JSON.stringify({ username: state.auth.username, amount: 100 }),
    });
    await refreshBalance();
    showToast("Баланс пополнен на $100.00.");
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
  } finally {
    setLoading(false);
  }
}

function handleTileClick(index) {
  if (state.mode === "auto") {
    if (state.autoRunning) return;
    if (state.selected.has(index)) state.selected.delete(index);
    else if (state.selected.size < getDiamonds()) state.selected.add(index);
    else showNotice(`Максимум плиток: ${getDiamonds()}`);
    playSound("click");
    applySelection();
    return;
  }

  if (!state.game || state.game.status !== "inProcess") {
    showNotice("Сначала сделайте ставку");
    return;
  }
  if (!isTilePlayable(index)) return;

  playSound("click");
  void openCell(index);
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function runAutoBet() {
  if (state.selected.size === 0) {
    showNotice("Выберите плитки для автоигры");
    return;
  }

  const rounds = Number.parseInt(elements.autoRounds.value, 10);
  const total = Number.isFinite(rounds) && rounds > 0 ? rounds : Infinity;

  state.autoRunning = true;
  renderControls();

  for (let round = 0; round < total && state.autoRunning; round += 1) {
    const started = await startGame();
    if (!started || !state.autoRunning) break;

    let lost = false;
    for (const index of state.selected) {
      if (!state.autoRunning) break;
      const move = await openCell(index);
      if (!move) {
        lost = true;
        break;
      }
      if (move.result === "bomb") {
        lost = true;
        break;
      }
      await delay(180);
    }

    if (!state.autoRunning) break;
    if (!lost) await cashOut();

    state.game = null;
    renderControls();
    await delay(900);
    if (!state.autoRunning) break;
    resetBoard();
  }

  state.autoRunning = false;
  state.game = null;
  renderControls();
}

function handlePrimaryAction() {
  if (!state.auth) {
    showAuth();
    return;
  }

  if (state.mode === "auto") {
    if (state.autoRunning) {
      state.autoRunning = false;
      renderControls();
      return;
    }
    void runAutoBet();
    return;
  }

  if (state.game && state.game.status === "inProcess") {
    void cashOut().then(() => {
      state.game = null;
      renderControls();
    });
    return;
  }

  void startGame().then(() => renderControls());
}

function updateGridSize(gridSize) {
  state.gridSize = gridSize;
  localStorage.setItem(gridSizeKey, String(gridSize));
  state.mines = Math.min(Math.max(state.mines, 1), gridSize - 1);
  state.selected.clear();
  state.game = null;
  resetBoard();
  renderControls();
}

function updateGemCount(value) {
  const gems = Number.parseInt(value, 10);
  if (!Number.isFinite(gems)) return;
  const clamped = Math.min(Math.max(gems, 1), state.gridSize - 1);
  state.mines = state.gridSize - clamped;
  state.selected.clear();
  applySelection();
  renderControls();
}

function switchMode(mode) {
  if (state.mode === mode || state.autoRunning) return;
  state.mode = mode;
  state.selected.clear();
  state.game = null;
  resetBoard();
  renderControls();
  if (mode === "auto") showNotice("Выберите плитки для автоигры");
}

function signOut() {
  if (state.loading || state.autoRunning) return;
  localStorage.removeItem(storageKey);
  state.auth = null;
  state.balance = null;
  state.game = null;
  elements.authForm.reset();
  resetBoard();
  renderAccount();
  renderControls();
  showAuth();
}

elements.gridOptions.addEventListener("click", (event) => {
  const option = event.target.closest("[data-grid-size]");
  if (!option || option.disabled) return;
  updateGridSize(Number(option.dataset.gridSize));
});

elements.betMode.addEventListener("click", (event) => {
  const tab = event.target.closest("[data-mode]");
  if (!tab || tab.disabled) return;
  switchMode(tab.dataset.mode);
});

elements.mineCount.addEventListener("input", (event) => updateGemCount(event.target.value));

function gemCountFromPointer(event) {
  const bar = elements.minesBar.getBoundingClientRect();
  if (!bar.width) return null;
  const fraction = Math.min(Math.max((event.clientX - bar.left) / bar.width, 0), 1);
  const maxGems = state.gridSize - 1;
  return Math.round(1 + fraction * (maxGems - 1));
}

let draggingSlider = false;

elements.minesTrack.addEventListener("pointerdown", (event) => {
  if (elements.mineCount.disabled) return;
  const gems = gemCountFromPointer(event);
  if (gems === null) return;
  draggingSlider = true;
  elements.minesTrack.setPointerCapture(event.pointerId);
  event.preventDefault();
  updateGemCount(gems);
});

elements.minesTrack.addEventListener("pointermove", (event) => {
  if (!draggingSlider) return;
  const gems = gemCountFromPointer(event);
  if (gems === null) return;
  event.preventDefault();
  updateGemCount(gems);
});

function stopSliderDrag(event) {
  if (!draggingSlider) return;
  draggingSlider = false;
  if (elements.minesTrack.hasPointerCapture(event.pointerId)) {
    elements.minesTrack.releasePointerCapture(event.pointerId);
  }
}

elements.minesTrack.addEventListener("pointerup", stopSliderDrag);
elements.minesTrack.addEventListener("pointercancel", stopSliderDrag);

elements.betAmount.addEventListener("change", (event) => {
  state.betAmount = event.target.value;
  setError();
});

elements.betHalf.addEventListener("click", () => {
  const amount = Number(elements.betAmount.value.replace(",", ".")) || 0;
  state.betAmount = Math.max(amount / 2, 0).toFixed(2);
  renderControls();
});

elements.betDouble.addEventListener("click", () => {
  const amount = Number(elements.betAmount.value.replace(",", ".")) || 0;
  state.betAmount = (amount * 2).toFixed(2);
  renderControls();
});

elements.placeBet.addEventListener("click", handlePrimaryAction);
elements.topUp.addEventListener("click", topUpBalance);
elements.changeUser.addEventListener("click", () => {
  if (state.auth) signOut();
  else showAuth();
});

elements.soundToggle.addEventListener("click", () => {
  state.soundOn = !state.soundOn;
  localStorage.setItem(soundKey, state.soundOn ? "on" : "off");
  elements.soundToggle.classList.toggle("is-muted", !state.soundOn);
});

elements.winDisplay.addEventListener("click", () => elements.winDisplay.classList.add("is-hidden"));

elements.authForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const username = elements.authUsername.value.trim();
  const password = elements.authPassword.value;
  if (!username || !password) {
    elements.authError.textContent = "Введите логин и пароль.";
    return;
  }
  saveAuth({ username, password });
  hideAuth();
  void refreshBalance().then(renderControls);
});

preloadAssets();
elements.soundToggle.classList.toggle("is-muted", !state.soundOn);
resetBoard();
renderAccount();
renderControls();
void refreshBalance().then(renderControls);
