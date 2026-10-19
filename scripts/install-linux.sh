#!/usr/bin/env bash
# Instalador eToolsCode (Linux)
set -euo pipefail

APP_NAME="etoolscode"
INSTALL_DIR="${HOME}/.local/share/etoolscode"
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.ds-ac"

echo "=== eToolsCode — instalador Linux ==="

mkdir -p "$INSTALL_DIR" "$BIN_DIR" "$CONFIG_DIR"

# Copia binário do diretório atual ou ARG1
SRC_BIN="${1:-./etoolscode}"
if [ ! -f "$SRC_BIN" ]; then
  echo "Uso: $0 /caminho/para/etoolscode"
  echo "Binário não encontrado: $SRC_BIN"
  exit 1
fi
cp -f "$SRC_BIN" "$INSTALL_DIR/$APP_NAME"
chmod +x "$INSTALL_DIR/$APP_NAME"
ln -sfn "$INSTALL_DIR/$APP_NAME" "$BIN_DIR/$APP_NAME"

# Senha da UI
echo ""
echo "Defina a senha de acesso à interface web:"
read -r -s -p "Senha: " PASS1; echo
read -r -s -p "Confirme: " PASS2; echo
if [ -z "$PASS1" ] || [ "$PASS1" != "$PASS2" ]; then
  echo "Senhas não conferem."
  exit 1
fi

HASH=$("$INSTALL_DIR/$APP_NAME" --hash-password "$PASS1" 2>/dev/null || true)
if [ -z "$HASH" ]; then
  # fallback bcrypt via openssl não é bcrypt; grava plain só se hash falhar — preferir hash
  if command -v htpasswd >/dev/null 2>&1; then
    HASH=$(htpasswd -nbBC 10 "" "$PASS1" | tr -d ':\n' | sed 's/^//')
  else
    echo "AVISO: não foi possível gerar bcrypt. Defina DSAC_UI_PASSWORD_HASH manualmente."
    HASH=""
  fi
fi

ENV_FILE="$INSTALL_DIR/.env"
cat > "$ENV_FILE" <<EOF
HTTP_PORT=8080
DSAC_UI_PASSWORD_HASH=$HASH
ISSUES_DIR=$HOME/etoolscode-data/issues
LLM_CLIENT=1
DEEPSEEK_API_KEY=
META_API_KEY=
GEMINI_API_KEY=
EOF

mkdir -p "$HOME/etoolscode-data/issues"
echo '{"issues_dir":"'"$HOME/etoolscode-data/issues"'","projects_root":"'"$HOME"'","projects":[]}' > "$CONFIG_DIR/workspace.json"

echo ""
echo "Instalado."
echo "  Binário: $BIN_DIR/$APP_NAME"
echo "  Config:  $ENV_FILE"
echo "  Dados:   $HOME/etoolscode-data"
echo ""
echo "1) Edite API keys: nano $ENV_FILE"
echo "  ou use ⚙ Config → Modelos na UI"
echo "2) Inicie: cd $INSTALL_DIR && $APP_NAME --ui"
echo "3) Abra:   http://127.0.0.1:8080"
