const gridSizes = [25, 36, 49, 64];
const storageKey = "rainbet-mines-auth";
const bombFrames = [
  "/assets/mine/2.webp",
  "/assets/mine/7.webp",
  "/assets/mine/14.webp",
  "/assets/mine/22.webp",
  "/assets/mine/26.webp",
];

const state = {
  auth: loadAuth(),
  game: null,
  gridSize: 25,
  mines: 3,
  betAmount: "10.00",
  balance: null,
  loading: false,
};

const elements = {
  authError: document.querySelector("#auth-error"),
  authForm: document.querySelector("#auth-form"),
  authOverlay: document.querySelector("#auth-overlay"),
  authPassword: document.querySelector("#auth-password"),
  authUsername: document.querySelector("#auth-username"),
  accountName: document.querySelector("#account-name"),
  betAmount: document.querySelector("#bet-amount"),
  balanceValue: document.querySelector("#balance-value"),
  board: document.querySelector("#mines-board"),
  boardStatus: document.querySelector("#board-status"),
  cashout: document.querySelector("#cashout"),
  cashoutValue: document.querySelector("#cashout-value"),
  changeUser: document.querySelector("#change-user"),
  footerCopy: document.querySelector("#footer-copy"),
  gridOptions: document.querySelector("#grid-options"),
  inputError: document.querySelector("#input-error"),
  mineCount: document.querySelector("#mine-count"),
  mineDecrease: document.querySelector("#mine-decrease"),
  mineIncrease: document.querySelector("#mine-increase"),
  multiplier: document.querySelector("#multiplier-display"),
  oddsNote: document.querySelector("#odds-note"),
  placeBet: document.querySelector("#place-bet"),
  placeBetLabel: document.querySelector("#place-bet-label"),
  topUp: document.querySelector("#top-up"),
  toast: document.querySelector("#toast"),
};

let bombAnimationTimer;
let toastTimer;

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

function getBoardColumns() {
  return Math.sqrt(state.gridSize);
}

function getDiamonds() {
  return state.gridSize - state.mines;
}

function setError(message = "") {
  elements.inputError.textContent = message;
}

function showToast(message, isError = false) {
  clearTimeout(toastTimer);
  elements.toast.textContent = message;
  elements.toast.classList.toggle("error", isError);
  elements.toast.classList.add("show");
  toastTimer = setTimeout(() => elements.toast.classList.remove("show"), 3600);
}

function stopBombAnimation() {
  clearTimeout(bombAnimationTimer);
  bombAnimationTimer = null;
}

function playBombAnimation(gameID) {
  const frameOrder = [0, 1, 2, 3, 4, 0];
  stopBombAnimation();

  function showFrame(orderIndex) {
    if (!state.game || state.game.id !== gameID || state.game.status !== "failed") return;

    const bomb = elements.board.querySelector(".bomb-frame");
    if (!bomb) return;

    bomb.src = bombFrames[frameOrder[orderIndex]];
    if (orderIndex === frameOrder.length - 1) return;

    const delay = orderIndex === frameOrder.length - 2 ? 350 : 110;
    bombAnimationTimer = setTimeout(() => showFrame(orderIndex + 1), delay);
  }

  showFrame(0);
}

function showAuth(message = "Your session needs authorization.") {
  elements.authOverlay.classList.remove("is-hidden");
  elements.authError.textContent = message;
  elements.authUsername.focus();
}

function hideAuth() {
  elements.authOverlay.classList.add("is-hidden");
  elements.authError.textContent = "";
}

function formatCurrency(value) {
  return `$${value}`;
}

function renderAccount() {
  const username = state.auth && state.auth.username;
  elements.accountName.textContent = username || "Not signed in";
  elements.balanceValue.textContent = state.balance === null ? "—" : formatCurrency(state.balance);
  elements.changeUser.textContent = username ? "Switch user" : "Sign in";
}

function renderControls() {
  elements.betAmount.value = state.betAmount;
  elements.mineCount.value = String(state.mines);
  elements.oddsNote.textContent = `${getDiamonds()} diamonds · ${state.mines} mines`;

  [...elements.gridOptions.querySelectorAll(".option")].forEach((option) => {
    const active = Number(option.dataset.gridSize) === state.gridSize;
    option.classList.toggle("active", active);
    option.setAttribute("aria-checked", String(active));
  });

  const active = Boolean(state.game && state.game.status === "inProcess");
  elements.betAmount.disabled = active || state.loading;
  elements.mineCount.disabled = active || state.loading;
  elements.mineDecrease.disabled = active || state.loading;
  elements.mineIncrease.disabled = active || state.loading;
  [...elements.gridOptions.querySelectorAll(".option")].forEach((option) => {
    option.disabled = active || state.loading;
  });
  elements.placeBet.disabled = state.loading || active;
  elements.placeBetLabel.textContent = state.game && state.game.status !== "inProcess" ? "Play again" : "Place bet";
  elements.changeUser.disabled = active || state.loading;
  elements.topUp.disabled = state.loading;
}

function renderBoard() {
  const activeGame = state.game && state.game.status === "inProcess";
  const openedCells = (state.game && state.game.openedCells) || [];
  const lastMove = state.game && state.game.lastMove;

  elements.board.style.setProperty("--columns", getBoardColumns());
  elements.board.replaceChildren();

  for (let index = 0; index < state.gridSize; index += 1) {
    const cell = document.createElement("button");
    const opened = openedCells.includes(index);
    cell.className = "tile";
    cell.type = "button";
    cell.setAttribute("role", "gridcell");
    cell.setAttribute("aria-label", `Cell ${index + 1}`);
    cell.disabled = !activeGame || opened || state.loading;

    if (opened) {
      const isBomb = lastMove && lastMove.result === "bomb" && lastMove.cellIndex === index;
      cell.classList.add(isBomb ? "open-bomb" : "open-diamond");
      if (isBomb) {
        const bomb = document.createElement("img");
        bomb.className = "bomb-frame";
        bomb.src = bombFrames[0];
        bomb.alt = "Mine";
        cell.append(bomb);
      } else {
        const diamond = document.createElement("img");
        diamond.className = "tile-icon";
        diamond.src = "/assets/diamond-final.png";
        diamond.alt = "Diamond";
        cell.append(diamond);
      }
    }

    cell.addEventListener("click", () => openCell(index));
    elements.board.append(cell);
  }

  if (!state.game) {
    elements.boardStatus.textContent = "Choose your wager to start.";
    elements.footerCopy.textContent = `${state.gridSize} cells ready`;
    elements.multiplier.textContent = "—";
    elements.cashout.classList.add("is-hidden");
  } else if (state.game.status === "inProcess") {
    const openCount = openedCells.length;
    elements.boardStatus.textContent = openCount ? "Diamond found. Continue or cash out." : "Pick a tile to reveal your first diamond.";
    elements.footerCopy.textContent = `${openCount} diamond${openCount === 1 ? "" : "s"} opened`;
    elements.multiplier.textContent = state.game.multiplier ? `${state.game.multiplier}×` : "—";

    if (openCount > 0) {
      elements.cashout.classList.remove("is-hidden");
      elements.cashoutValue.textContent = state.game.multiplier ? `${state.game.multiplier}×` : "Cash out";
    } else {
      elements.cashout.classList.add("is-hidden");
    }
  } else if (state.game.status === "failed") {
    elements.boardStatus.textContent = "Mine hit. The game is over.";
    elements.footerCopy.textContent = "Your wager has been settled";
    elements.multiplier.textContent = "0×";
    elements.cashout.classList.add("is-hidden");
  } else {
    elements.boardStatus.textContent = `Cashed out ${formatCurrency(state.game.payout)}.`;
    elements.footerCopy.textContent = "Payout sent to your balance";
    elements.multiplier.textContent = `${state.game.multiplier}×`;
    elements.cashout.classList.add("is-hidden");
  }
}

function setLoading(loading) {
  state.loading = loading;
  renderControls();
  renderBoard();
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
    showAuth();
    throw new Error("Authentication required");
  }
  if (!response.ok) {
    throw new Error(body.error || "The request could not be completed.");
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

async function startGame() {
  setError();
  if (state.game && state.game.status === "inProcess") return;
  stopBombAnimation();

  const amount = elements.betAmount.value.trim();
  if (!amount || Number(amount) <= 0) {
    setError("Enter a bet amount greater than $0.00.");
    return;
  }

  state.betAmount = amount;
  setLoading(true);
  try {
    const game = await api("/mines/bets", {
      method: "POST",
      body: JSON.stringify({
        betAmount: amount,
        gridSize: state.gridSize,
        mines: state.mines,
        demo: false,
        clientSeed: newClientSeed(),
      }),
    });

    state.game = {
      id: game.id,
      status: game.status,
      openedCells: [],
      multiplier: null,
      lastMove: null,
      payout: null,
    };
    showToast(`Game #${game.id} started. Choose a tile.`);
    void refreshBalance();
  } catch (error) {
    if (error.message !== "Authentication required") setError(error.message);
  } finally {
    setLoading(false);
  }
}

async function openCell(cellIndex) {
  if (!state.game || state.game.status !== "inProcess" || state.loading) return;

  let bombOpened = false;
  setLoading(true);
  try {
    const move = await api(`/mines/bets/${state.game.id}/moves`, {
      method: "POST",
      body: JSON.stringify({ cellIndex }),
    });

    state.game.status = move.status;
    state.game.openedCells = move.openedCells;
    state.game.multiplier = move.multiplier || null;
    state.game.lastMove = { cellIndex, result: move.result };
    bombOpened = move.result === "bomb";
    showToast(move.result === "bomb" ? "A mine was revealed." : "Diamond revealed.", move.result === "bomb");
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
  } finally {
    setLoading(false);
    if (bombOpened && state.game) playBombAnimation(state.game.id);
  }
}

async function cashOut() {
  if (!state.game || state.game.status !== "inProcess" || !state.game.openedCells.length || state.loading) return;

  setLoading(true);
  try {
    const cashout = await api(`/mines/bets/${state.game.id}/cashout`, { method: "POST", body: "{}" });
    state.game.status = cashout.status;
    state.game.payout = cashout.payout;
    state.game.multiplier = cashout.multiplier;
    showToast(`Cashed out ${formatCurrency(cashout.payout)}.`);
    void refreshBalance();
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
  } finally {
    setLoading(false);
  }
}

async function topUpBalance() {
  if (!state.auth) {
    showAuth("Sign in before topping up.");
    return;
  }

  setLoading(true);
  try {
    await api("/user/balance", {
      method: "POST",
      body: JSON.stringify({
        username: state.auth.username,
        amount: Number(elements.betAmount.value),
      }),
    });
    await refreshBalance();
    showToast("Balance topped up.");
  } catch (error) {
    if (error.message !== "Authentication required") showToast(error.message, true);
  } finally {
    setLoading(false);
  }
}

function updateGridSize(gridSize) {
  if (state.game && state.game.status !== "inProcess") {
    stopBombAnimation();
    state.game = null;
  }
  state.gridSize = gridSize;
  state.mines = Math.min(Math.max(state.mines, 1), gridSize - 1);
  renderControls();
  renderBoard();
}

function updateMineCount(value) {
  const count = Number.parseInt(value, 10);
  if (!Number.isFinite(count)) return;
  state.mines = Math.min(Math.max(count, 1), state.gridSize - 1);
  renderControls();
  renderBoard();
}

function switchUser() {
  if (state.loading || (state.game && state.game.status === "inProcess")) return;

  localStorage.removeItem(storageKey);
  stopBombAnimation();
  state.auth = null;
  state.balance = null;
  state.game = null;
  elements.authForm.reset();
  renderAccount();
  renderControls();
  renderBoard();
  showAuth("Sign in with another user.");
}

elements.gridOptions.addEventListener("click", (event) => {
  const option = event.target.closest("[data-grid-size]");
  const gameInProgress = state.game && state.game.status === "inProcess";
  if (option && !gameInProgress && !state.loading) updateGridSize(Number(option.dataset.gridSize));
});
elements.mineDecrease.addEventListener("click", () => updateMineCount(state.mines - 1));
elements.mineIncrease.addEventListener("click", () => updateMineCount(state.mines + 1));
elements.mineCount.addEventListener("change", (event) => updateMineCount(event.target.value));
elements.betAmount.addEventListener("change", (event) => { state.betAmount = event.target.value; });
elements.placeBet.addEventListener("click", startGame);
elements.cashout.addEventListener("click", cashOut);
elements.topUp.addEventListener("click", topUpBalance);
elements.changeUser.addEventListener("click", switchUser);
elements.authForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const username = elements.authUsername.value.trim();
  const password = elements.authPassword.value;
  if (!username || !password) {
    elements.authError.textContent = "Enter both username and password.";
    return;
  }
  saveAuth({ username, password });
  hideAuth();
  void refreshBalance();
  showToast("Credentials saved. Retry your action to continue.");
});

renderAccount();
renderControls();
renderBoard();
void refreshBalance();
