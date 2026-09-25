// Classificação da média diária (peso vivo, em kg) — ajuste conforme sua operação.
const LIMITE_MAGRO = 400; // abaixo disso: magro
const LIMITE_GORDO = 480; // a partir disso: gordo
const RENDIMENTO_CARCACA = 0.5; // 50% para conversão em arrobas (@ = kg vivo * rendimento / 15)

const $ = (id) => document.getElementById(id);

function hojeISO() {
  return new Date().toLocaleDateString("sv-SE"); // YYYY-MM-DD no fuso local
}

function fmtKg(v) {
  return Number(v).toLocaleString("pt-BR", { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

function arrobas(kg) {
  return (kg * RENDIMENTO_CARCACA) / 15;
}

function classificar(mediaKg) {
  if (mediaKg < LIMITE_MAGRO) return { texto: "Magro", classe: "magro" };
  if (mediaKg < LIMITE_GORDO) return { texto: "Médio", classe: "medio" };
  return { texto: "Gordo", classe: "gordo" };
}

async function fetchJSON(url, opts = {}) {
  opts.headers = opts.headers || {};
  const chave = localStorage.getItem("apiKey");
  if (chave) opts.headers["X-API-Key"] = chave;
  const resp = await fetch(url, opts);
  const body = await resp.json().catch(() => ({}));
  if (!resp.ok) {
    const err = new Error(body.erro || "Erro na requisição");
    err.status = resp.status;
    throw err;
  }
  return body;
}

function renderResumo(r) {
  const temDados = r.total_pesagens > 0;

  $("totalPesagens").textContent = r.total_pesagens;
  $("mediaPeso").textContent = temDados ? fmtKg(r.media_peso_kg) : "—";
  $("maiorPeso").textContent = temDados ? fmtKg(r.maior_peso_kg) : "—";
  $("menorPeso").textContent = temDados ? fmtKg(r.menor_peso_kg) : "—";
  $("mediaArroba").textContent = temDados ? `≈ ${arrobas(r.media_peso_kg).toFixed(1)} @` : "";

  const badge = $("condicao");
  badge.className = "badge";
  if (temDados) {
    const c = classificar(r.media_peso_kg);
    badge.textContent = c.texto;
    badge.classList.add(c.classe);
  } else {
    badge.textContent = "sem dados";
  }

  renderGrafico(r.media_por_horario);
}

function renderGrafico(horarios) {
  const grafico = $("grafico");
  const vazio = $("graficoVazio");
  grafico.innerHTML = "";

  if (!horarios || horarios.length === 0) {
    vazio.hidden = false;
    return;
  }
  vazio.hidden = true;

  const max = Math.max(...horarios.map((h) => h.media_kg));
  for (const h of horarios) {
    const grupo = document.createElement("div");
    grupo.className = "barra-grupo";
    grupo.title = `${h.hora}h — média ${fmtKg(h.media_kg)} kg (${h.total} pesagem${h.total > 1 ? "s" : ""})`;

    const valor = document.createElement("span");
    valor.className = "barra-valor";
    valor.textContent = fmtKg(h.media_kg);

    const barra = document.createElement("div");
    barra.className = "barra";
    barra.style.height = `${Math.max(4, (h.media_kg / max) * 100)}%`;

    const hora = document.createElement("span");
    hora.className = "barra-hora";
    hora.textContent = `${h.hora}h`;

    grupo.append(valor, barra, hora);
    grafico.appendChild(grupo);
  }
}

function renderTabela(itens) {
  const tbody = $("tabelaPesagens");
  const vazio = $("tabelaVazia");
  tbody.innerHTML = "";

  if (!itens || itens.length === 0) {
    vazio.hidden = false;
    return;
  }
  vazio.hidden = true;

  for (const p of itens.slice(0, 12)) {
    const tr = document.createElement("tr");
    const hora = new Date(p.data_hora).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
    tr.innerHTML = `
      <td>${hora}</td>
      <td>${p.brinco ? escapeHTML(p.brinco) : "<span style='color:var(--texto-suave)'>—</span>"}</td>
      <td class="num">${fmtKg(p.peso_kg)}</td>
      <td class="num">${arrobas(p.peso_kg).toFixed(1)}</td>`;
    tbody.appendChild(tr);
  }
}

function escapeHTML(s) {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

async function carregar() {
  const data = $("data").value || hojeISO();
  try {
    const [resumo, pesagens] = await Promise.all([
      fetchJSON(`/api/resumo?data=${data}`),
      fetchJSON(`/api/pesagens?data=${data}`),
    ]);
    renderResumo(resumo);
    renderTabela(pesagens);
  } catch (e) {
    console.error("Erro ao carregar dashboard:", e);
  }
}

function mostrarMsg(texto, ok) {
  const msg = $("msgForm");
  msg.textContent = texto;
  msg.className = `msg ${ok ? "ok" : "erro"}`;
  msg.hidden = false;
  setTimeout(() => (msg.hidden = true), 4000);
}

$("formPesagem").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const peso = parseFloat($("peso").value.replace(",", "."));
  if (!peso || peso <= 0) {
    mostrarMsg("Informe um peso válido.", false);
    return;
  }
  try {
    await enviarPesagem(peso);
    mostrarMsg("Pesagem registrada com sucesso!", true);
    $("formPesagem").reset();
    $("data").value = hojeISO();
    carregar();
  } catch (e) {
    if (e.status === 401) {
      const chave = prompt("Este servidor exige chave de API para registrar pesagens.\nInforme a chave:");
      if (chave && chave.trim()) {
        localStorage.setItem("apiKey", chave.trim());
        try {
          await enviarPesagem(peso);
          mostrarMsg("Pesagem registrada com sucesso!", true);
          $("formPesagem").reset();
          carregar();
          return;
        } catch (e2) {
          mostrarMsg(e2.message, false);
          return;
        }
      }
    }
    mostrarMsg(e.message, false);
  }
});

function enviarPesagem(peso) {
  return fetchJSON("/api/pesagens", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ peso_kg: peso, brinco: $("brinco").value.trim() }),
  });
}

$("data").addEventListener("change", carregar);
$("atualizar").addEventListener("click", carregar);

$("data").value = hojeISO();
carregar();
setInterval(carregar, 30000); // atualização automática
