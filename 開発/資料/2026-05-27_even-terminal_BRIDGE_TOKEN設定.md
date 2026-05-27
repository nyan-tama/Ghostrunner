# even-terminal BRIDGE_TOKEN 固定化メモ

## なぜ固定化したか

`even-terminal` は起動毎にランダムな 16 バイトのトークンを再生成する仕様
（`/opt/homebrew/lib/node_modules/@evenrealities/even-terminal/dist/index.js:11`）。

```js
const TOKEN = process.env.BRIDGE_TOKEN ?? randomBytes(16).toString("hex");
```

このためサーバーを再起動すると以下の問題が発生:

- スマホ（iPhone, Tailscale 経由）のブラウザにブックマークした URL のトークンが古くなり 401
- 再接続には QR を読み直す必要があるが、サーバー起動時に画面を出していないと QR が手に入らない
- 過去ログ（`/Users/user/Ghostrunner/even-terminal-2026-05-27T08-06-10-944Z.log`）でも同症状で大量に 401 が出ていた

→ `BRIDGE_TOKEN` 環境変数を固定して回避。スマホのブックマークが永続的に使える。

## トークンの保管場所

`~/.zshrc` の末尾に追記済み:

```bash
# even-terminal: 固定トークン (スマホブックマーク永続化のため)
export BRIDGE_TOKEN=<32桁hex>
```

確認コマンド:

```bash
grep BRIDGE_TOKEN ~/.zshrc
# または
echo $BRIDGE_TOKEN   # 新規ターミナルで
```

**トークン値そのものはこのファイルに記載しない**（漏洩リスク回避）。
必要なら上記コマンドで取得する。

## 起動・停止

### 起動（フォアグラウンド推奨）

```bash
cd /Users/user/Ghostrunner
even-terminal --tailscale --provider claude
```

`~/.zshrc` の `BRIDGE_TOKEN` が自動で効くので、ログイン済みターミナルなら追加設定は不要。

### 停止

`Ctrl+C`（フォアグラウンドの場合）または:

```bash
pkill -f "even-terminal --tailscale"
```

### 背景起動したいとき

```bash
cd /Users/user/Ghostrunner
nohup even-terminal --tailscale --provider claude > /dev/null 2>&1 &
```

ログは `./even-terminal-<タイムスタンプ>.log` に自動で残る。

## 接続URL

```
http://<host>:3456?token=<BRIDGE_TOKEN>&defaultProvider=claude
```

| 端末 | host |
|------|------|
| Mac 本体 | `localhost` |
| スマホ（Tailscale 経由） | `100.68.245.31`（このMacの Tailscale IPv4） |

URL を毎回組み立てる用ワンライナー:

```bash
echo "http://$(tailscale ip -4):3456?token=${BRIDGE_TOKEN}&defaultProvider=claude"
```

## QR コードを後から再表示する

通常はサーバー起動時のバナーで表示されるが、後で読みたくなった場合:

```bash
node -e "
const qr = require('/opt/homebrew/lib/node_modules/@evenrealities/even-terminal/node_modules/qrcode-terminal');
const url = \`http://\${process.env.TS_IP}:3456?token=\${process.env.BRIDGE_TOKEN}&defaultProvider=claude\`;
qr.generate(url, {small: true});
" TS_IP=$(tailscale ip -4)
```

## トラブルシュート

### スマホで 401 が出る

1. `~/.zshrc` の `BRIDGE_TOKEN` と、ブラウザ URL の `token=` が一致しているか確認
2. ずれていたら QR を読み直す（上記「QR コードを後から再表示する」）
3. `~/.zshrc` を編集した直後は、起動中の `even-terminal` を再起動しないと新トークンは効かない

### トークンをローテーションしたいとき

```bash
# 新トークン生成
NEW=$(openssl rand -hex 16)

# ~/.zshrc の BRIDGE_TOKEN 行を置換（macOS の sed）
sed -i '' "s|^export BRIDGE_TOKEN=.*|export BRIDGE_TOKEN=$NEW|" ~/.zshrc

# even-terminal を再起動
pkill -f "even-terminal --tailscale"
source ~/.zshrc
cd /Users/user/Ghostrunner
even-terminal --tailscale --provider claude
```

その後スマホでも QR を読み直す。

## 関連

- 過去の事故ログ: `/Users/user/Ghostrunner/even-terminal-2026-05-27T08-06-10-944Z.log`
  - PC は新トークン、スマホは旧トークンで全リクエスト 401 だった
- 該当ソース: `/opt/homebrew/lib/node_modules/@evenrealities/even-terminal/dist/index.js`
- CLI 起動オプション: `even-terminal --help`
