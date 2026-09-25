# Instala (ou remove) a inicializacao automatica do agente da balanca.
# Execute NO PC DO CLIENTE (o PC conectado a balanca), na pasta do agente:
#
#   powershell -ExecutionPolicy Bypass -File .\instalar-inicializacao.ps1
#   powershell -ExecutionPolicy Bypass -File .\instalar-inicializacao.ps1 -Desinstalar
#
# O agente inicia junto com o Windows, invisivel (sem janela), e grava tudo
# em agente.log na mesma pasta.

param(
    [switch]$Desinstalar
)

$ErrorActionPreference = "Stop"
$nomeTarefa = "AgenteBalancaFrigorifico"
$pasta = $PSScriptRoot
$exeOculto = Join-Path $pasta "agente-balanca-oculto.exe"
$atalho = Join-Path ([Environment]::GetFolderPath("Startup")) "AgenteBalanca.lnk"

function Parar-Agente {
    Get-Process -Name "agente-balanca", "agente-balanca-oculto" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-Process -Id $_.Id -Force }
}

if ($Desinstalar) {
    cmd /c "schtasks /End /TN $nomeTarefa >nul 2>&1"
    cmd /c "schtasks /Delete /TN $nomeTarefa /F >nul 2>&1"
    if (Test-Path $atalho) { Remove-Item $atalho }
    Parar-Agente
    Write-Host "Inicializacao automatica removida." -ForegroundColor Green
    return
}

if (-not (Test-Path $exeOculto)) {
    Write-Host "ERRO: agente-balanca-oculto.exe nao encontrado nesta pasta." -ForegroundColor Red
    Write-Host "Copie o executavel para ca ou compile com:" -ForegroundColor Red
    Write-Host '  go build -ldflags "-H windowsgui" -o agente-balanca-oculto.exe .\agente' -ForegroundColor Red
    exit 1
}

# Plano A: tarefa agendada no logon (permite reinicio automatico via Agendador).
cmd /c "schtasks /Delete /TN $nomeTarefa /F >nul 2>&1"
cmd /c "schtasks /Create /TN $nomeTarefa /SC ONLOGON /RL LIMITED /TR `"\`"$exeOculto\`"`" /F >nul 2>&1"
if ($LASTEXITCODE -eq 0) {
    Write-Host "Tarefa agendada '$nomeTarefa' criada (inicia no logon do Windows)." -ForegroundColor Green
    cmd /c "schtasks /Run /TN $nomeTarefa >nul 2>&1"
} else {
    # Plano B: atalho na pasta Inicializar do usuario (nao exige permissao).
    $ws = New-Object -ComObject WScript.Shell
    $lnk = $ws.CreateShortcut($atalho)
    $lnk.TargetPath = $exeOculto
    $lnk.WorkingDirectory = $pasta
    $lnk.Description = "Agente da balanca do frigorifico"
    $lnk.Save()
    Write-Host "Atalho criado na pasta Inicializar (inicia no logon do Windows)." -ForegroundColor Green
    Start-Process -FilePath $exeOculto -WorkingDirectory $pasta
}

Start-Sleep 4
$proc = Get-Process -Name "agente-balanca-oculto" -ErrorAction SilentlyContinue
if ($proc) {
    Write-Host "Agente em execucao, invisivel (PID $($proc.Id))." -ForegroundColor Green
    Write-Host "Acompanhe pelo arquivo agente.log nesta pasta."
} else {
    Write-Host "AVISO: agente ainda nao subiu - verifique o agente.log em $pasta" -ForegroundColor Yellow
}
