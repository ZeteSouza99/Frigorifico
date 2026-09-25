# Painel de Pesagem — Frigorífico 🐂

Dashboard 100% web para acompanhamento das pesagens do gado.
Backend em **Go**, banco de dados **Neon (PostgreSQL)** e frontend embutido no binário.

## Indicadores do dashboard

- **Média de peso do dia** — com conversão para arrobas e classificação (Magro / Médio / Gordo)
- **Número de pesagens** do dia
- **Maior e menor peso** do dia
- **Média de pesagem por horário** (gráfico de barras)
- Bônus: formulário para registrar pesagens e tabela com os últimos registros

## Como rodar

### 1. Modo demonstração (sem banco)

Sem `DATABASE_URL` definido, o sistema roda com dados de exemplo em memória:

```powershell
go run .
```

Abra: http://localhost:8080

### 2. Com Neon (PostgreSQL)

1. Crie uma conta/projeto gratuito em https://neon.tech
2. Copie a **connection string** do painel do Neon (algo como
   `postgresql://usuario:senha@ep-xxx-xxx.sa-east-1.aws.neon.tech/neondb?sslmode=require`)
3. Defina a variável de ambiente e rode:

```powershell
$env:DATABASE_URL = "postgresql://usuario:senha@ep-xxx.aws.neon.tech/neondb?sslmode=require"
go run .
```

O schema (`schema.sql`) é aplicado automaticamente na primeira execução.

### Variáveis de ambiente

| Variável       | Padrão              | Descrição                          |
|----------------|---------------------|------------------------------------|
| `DATABASE_URL` | *(vazio = modo demo)* | Connection string do Neon        |
| `PORT`         | `8080`              | Porta do servidor web (local)      |
| `APP_TZ`       | `America/Manaus`    | Fuso horário para agrupar por dia/hora |
| `API_KEY`      | *(vazio = POST aberto)* | Se definida, POST /api/pesagens exige header `X-API-Key` |

## API

| Método | Rota                      | Descrição                                  |
|--------|---------------------------|--------------------------------------------|
| GET    | `/api/resumo?data=YYYY-MM-DD`   | Indicadores do dia (média, total, maior/menor, média por hora) |
| GET    | `/api/pesagens?data=YYYY-MM-DD` | Lista as pesagens do dia             |
| POST   | `/api/pesagens`           | Registra pesagem: `{"peso_kg": 487.5, "brinco": "BR-25-014"}` |

## Deploy no Vercel

O projeto já está estruturado para o Vercel:
- `api/resumo/index.go` e `api/pesagens/index.go` → serverless functions em Go
- `public/` → frontend estático
- `internal/` → lógica compartilhada entre o servidor local e as functions

Passos:

```powershell
npm install -g vercel
vercel login
vercel                                   # primeiro deploy (preview)
vercel env add DATABASE_URL production   # cole a connection string do Neon
vercel env add APP_TZ production         # ex.: America/Manaus
vercel --prod                            # deploy de produção
```

Alternativa sem CLI: suba o projeto para o GitHub, importe em https://vercel.com/new
e configure as variáveis `DATABASE_URL` e `APP_TZ` em *Settings → Environment Variables*.

## Agente da balança (Toledo Tendal 200 / TI-200) + impressora Zebra

O agente (`agente/`) roda no PC ao lado da balança, lê o peso pela serial
RS-232, detecta pesagem estável, envia para a API (local ou Vercel) e imprime
etiqueta ZPL na Zebra. Sem internet, guarda em `pendentes.jsonl` e reenvia sozinho.

```powershell
go build -o agente-balanca.exe .\agente
.\agente-balanca.exe -listar             # descobre a porta COM da balança
.\agente-balanca.exe -sniffer            # mostra os bytes crus (conferir protocolo)
.\agente-balanca.exe -simular -ciclos 3  # testa tudo sem a balança
.\agente-balanca.exe -testar-impressao   # imprime etiqueta de teste na Zebra
.\agente-balanca.exe                     # produção
```

Na primeira execução são criados:
- **`agente.json`** — configuração:
  - `serial`: porta COM, baud (TI-200 padrão: 4800, 7 bits, paridade par),
    `protocolo`: `toledo_p03` (contínuo STX) ou `generico` (linhas de texto),
    `casas_decimais` (-1 = automático)
  - `captura`: peso mínimo, nº de leituras estáveis, tolerância e rearme
  - `api`: `url` (ex.: `https://seu-projeto.vercel.app`) e `chave` (API key)
  - `impressora`: `tipo` = `rede` (IP:9100), `windows` (nome da impressora USB),
    `arquivo` (gera .zpl p/ conferir em https://labelary.com/viewer.html) ou `desativada`
- **`etiqueta.zpl`** — template da etiqueta (10x5 cm). Placeholders:
  `{PESO}`, `{ARROBAS}`, `{DATA}`, `{HORA}`, `{CODIGO}`, `{BRINCO}`

Fluxo em produção: pendura o animal → peso estabiliza → agente registra na API
→ etiqueta sai na Zebra → dashboard atualiza em qualquer celular/PC.

### Implantação no PC do cliente (PC conectado à balança)

1. Copie para uma pasta (ex.: `C:\AgenteBalanca`):
   - `agente-balanca.exe` (console — usado para configurar/diagnosticar)
   - `agente-balanca-oculto.exe` (produção, roda invisível; compile com
     `go build -ldflags "-H windowsgui" -o agente-balanca-oculto.exe .\agente`)
   - `instalar-inicializacao.ps1`
2. Configure: rode `.\agente-balanca.exe -listar`, ajuste o `agente.json`
   (porta COM, URL da API no Vercel, chave, impressora) e confira com
   `-sniffer` / `-testar-impressao` / `-simular`.
3. Instale a inicialização automática:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\instalar-inicializacao.ps1
   ```
   O agente passa a iniciar junto com o Windows, invisível, gravando em `agente.log`.
   Para remover: mesmo comando com `-Desinstalar`.

Os arquivos `agente.json`, `etiqueta.zpl`, `agente.log` e `pendentes.jsonl`
ficam sempre na pasta do executável.

## Build para produção (local)

```powershell
go build -o frigorifico-dashboard.exe .
```

O executável é único e já contém o frontend (HTML/CSS/JS embutidos via `go:embed`).

## Classificação Magro/Gordo

Os limites ficam no início de `web/app.js` e podem ser ajustados:
- **Magro**: média < 400 kg
- **Médio**: 400–479 kg
- **Gordo**: ≥ 480 kg (≈ 16 @ com rendimento de carcaça de 50%)
