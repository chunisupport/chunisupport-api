# サーバー新設手順

APIサーバーを新設するときに必要な、ネットワークとセキュリティの設定をまとめます。アプリケーションの設定値は[設定ファイル・環境変数](configuration.md)、ログのローテーションは[logrotate設定手順](logrotate.md)を参照してください。

## 前提の構成

```text
利用者 → Cloudflare（プロキシ） → nginx（80, 443番） → API（127.0.0.1のアプリポート）
                                                       → MySQL（同一ホスト）
```

この構成では、次の2つが満たされて初めて、IP単位の制限が正しく働きます。

- APIが、Cloudflareの中継サーバーではなく利用者のIPを送信元として数えること
- 第三者が送信元IPを偽装できないこと（Cloudflareを迂回してnginxへ直接接続できないこと）

どちらかが欠けても、アプリケーションは正常に動いているように見えます。必ず[動作確認](#動作確認)まで実施してください。

以降の `api.example.com` は実際のホスト名に読み替えてください。

## 1. Cloudflare

- APIのホスト名をプロキシ有効（オレンジの雲）にします。プロキシが無効なホスト名は、手順4のファイアウォールで接続できなくなります。
- SSL/TLSの暗号化モードは「フル（厳密）」とし、nginxにはCloudflareのオリジン証明書を設定します。
- HTTP認証でLet's Encryptの証明書を取得する構成は、手順4と両立しません。DNS認証を使うか、オリジン証明書を使ってください。

## 2. nginx

### 実クライアントIPの復元

`/etc/nginx/conf.d/cloudflare-realip.conf` を、手順5のスクリプトで生成します。内容は次の形式です。

```nginx
set_real_ip_from <CloudflareのIP範囲>;  # 範囲ごとに1行
real_ip_header CF-Connecting-IP;
```

- この設定がないと、`$remote_addr` がCloudflareの中継サーバーのIPになります。同じ中継サーバーを通る利用者同士が、レート制限や一時保存の件数上限を共有してしまいます。
- `set_real_ip_from` に `0.0.0.0/0` や `::/0` を指定してはいけません。すべての接続元の `CF-Connecting-IP` を信頼することになり、直接接続できる場合は送信元を自由に偽装できます。
- `/etc/nginx/nginx.conf` が `conf.d/*.conf` を読み込むことと、`nginx -V` の出力に `http_realip_module` が含まれることを確認してください。

### APIへの転送

クライアントから持ち込まれた `X-Forwarded-For` を引き継がないよう、`$remote_addr` で上書きします。`$proxy_add_x_forwarded_for` は使いません。

```nginx
server {
    server_name api.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name api.example.com;

    ssl_certificate     /etc/nginx/ssl/api.example.com-origin.crt;
    ssl_certificate_key /etc/nginx/ssl/api.example.com-origin.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://127.0.0.1:<app_port>;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 3. アプリケーション

`.config/<APP_ENV>.settings.json` に、nginxのアドレスを信頼するプロキシとして設定します。未設定の場合はTCP接続元（nginx）が送信元になり、全利用者が1つの送信元として数えられます。

```json
{
  "client_ip": {
    "trusted_proxy_cidrs": ["127.0.0.1/32"]
  }
}
```

APIはループバックアドレスだけで待ち受け、外部へ直接公開しません。

`.env`、設定ファイル、Firebaseのサービスアカウント鍵などの秘密情報は、実行ユーザーだけが読める `0600` にします。

## 4. ファイアウォール

ufwを使い、次のとおり許可します。

| ポート | 接続元 |
|---|---|
| SSH | 必要な接続元（可能なら管理用のIPに限定） |
| 80, 443 | CloudflareのIP範囲のみ |
| MySQLなど、その他すべて | 許可しない |

ロックアウトを防ぐため、**有効化の前にSSHを許可**してください。80番と443番の許可は手順5のスクリプトが追加します。

```bash
sudo ufw default deny incoming
```

```bash
sudo ufw allow <SSHのポート>/tcp comment 'SSH'
```

```bash
sudo ufw enable
```

- IPv6のアドレスを持つサーバーでは、`/etc/default/ufw` が `IPV6=yes` であることを確認してください。
- クラウドの標準イメージには、`iptables` のルールがあらかじめ入っているものがあります。ufwと競合しないよう、`sudo iptables -S` で確認して整理してください。
- クラウド側のファイアウォール（セキュリティリストなど）も、同じ方針で設定します。OSとクラウドのどちらか一方の設定ミスで公開されないよう、両方で制限します。

### MySQL

外部から接続する必要がなければ、`bind-address = 127.0.0.1` と `mysqlx-bind-address = 127.0.0.1` を設定します。管理ツールからの接続にはSSHトンネルを使います。

## 5. CloudflareのIP範囲の自動更新

IP範囲が変わると、新しい範囲を経由したアクセスが拒否され、利用者のIPも正しく取得できなくなります。次のスクリプトで、公式の一覧に合わせて定期的に更新します。

`/usr/local/sbin/update-cloudflare-ips.sh` に置き、`sudo chown root:root` と `sudo chmod 750` を実行してください。

```bash
#!/usr/bin/env bash
# Cloudflare の IP 範囲に合わせて nginx の信頼元と ufw の許可元を更新する（root で実行）
set -euo pipefail
# cron の既定の PATH には nginx や ufw がある /usr/sbin が含まれない
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

v4=$(curl -fsS https://www.cloudflare.com/ips-v4)
v6=$(curl -fsS https://www.cloudflare.com/ips-v6)
# 取得に失敗した内容で全遮断しないよう、全行の形式を確認してから進める
if grep -qvE '^[0-9.]+/[0-9]+$' <<<"$v4" || grep -qvE '^[0-9a-f:]+/[0-9]+$' <<<"$v6"; then
  echo "Cloudflare の IP 範囲の形式が不正です" >&2
  exit 1
fi

conf=/etc/nginx/conf.d/cloudflare-realip.conf
had_conf=false
if [[ -f "$conf" ]]; then
  cp "$conf" "$conf.bak"
  had_conf=true
fi
{
  for r in $v4 $v6; do echo "set_real_ip_from $r;"; done
  echo "real_ip_header CF-Connecting-IP;"
} > "$conf"
if ! nginx -t; then
  if [[ "$had_conf" == true ]]; then
    cp "$conf.bak" "$conf"
  else
    rm -f "$conf"
  fi
  exit 1
fi
systemctl reload nginx

for r in $v4 $v6; do
  ufw allow from "$r" to any port 80,443 proto tcp comment 'Cloudflare'
done
# Cloudflare の一覧から外れた範囲の許可を消す
for r in $(ufw show added | awk "/^ufw allow from .* to any port 80,443 proto tcp comment 'Cloudflare'\$/{print \$4}"); do
  if ! grep -qxF "$r" <<<"$v4"$'\n'"$v6"; then
    ufw --force delete allow from "$r" to any port 80,443 proto tcp
  fi
done
```

一度手動で実行してから、週1回の実行を `/etc/cron.d/update-cloudflare-ips` に設定します。

```cron
# 日曜 18:00 UTC = 月曜 3:00 JST
0 18 * * 0 root /usr/local/sbin/update-cloudflare-ips.sh >> /var/log/update-cloudflare-ips.log 2>&1
```

### cronの時刻の注意

- cronは**サーバーのタイムゾーン**で動きます。`timedatectl` で確認してから時刻を決めてください。設定ファイルの `timezone` はcronに影響しません。上の例はUTCのサーバーを想定しています。
- 利用者の少ない時間帯を選び、既存のバッチと重ならないようにします（`crontab -l` と `/etc/cron.d/` を確認）。

## 6. セキュリティヘッダー

付与元は、ヘッダーごとに次のどちらか1か所だけにします。nginxでは付与しません。同じヘッダーを複数の層で付けると、重複した値をブラウザが無効として扱うことがあります。

| ヘッダー | 値 | 付与元 |
|---|---|---|
| `X-Content-Type-Options` | `nosniff` | アプリケーション |
| `X-Frame-Options` | `DENY` | アプリケーション |
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` | アプリケーション |
| `Referrer-Policy` | `no-referrer` | アプリケーション |
| `Strict-Transport-Security` | `max-age=15552000` | Cloudflare |

- アプリケーションは、エラーやメンテナンスの応答を含むすべての応答に付与します。値は `internal/info/info.go` で固定しており、環境ごとの設定はありません。APIはJSONだけを返すため、最も制限の強い値にしています。
- HSTSはTLSを終端するCloudflareで付与します。SSL/TLS → Edge Certificates → HTTP Strict Transport Security (HSTS) で、次のとおり設定します。
  - Max Age：6か月
  - Apply HSTS policy to subdomains (includeSubDomains)：オフ
  - Preload：オフ
  - No-Sniff Header：オフ（アプリケーションが付与するため）
- HSTSの設定はゾーン内のプロキシ有効なすべてのホスト名に適用され、有効にした後はブラウザが期限までHTTPでの接続を拒否します。ゾーン内のすべてのホスト名がHTTPSで応答できることを確認してから有効にしてください。
- `includeSubDomains` と `preload` は、ドメイン配下のすべてのホスト名をHTTPSに縛り、取り消しにも長い時間がかかるため、有効にしません。ステージングは共有ドメイン `f5.si` のサブドメインで、preloadは申請できず、`includeSubDomains` も不要です。
- nginxの設定には `add_header` でこれらのヘッダーを追加しないでください。Cloudflareの Transform Rules や Managed Transforms でも追加しないでください。

## 動作確認

新設時と、nginx、ファイアウォール、Cloudflareの設定、セキュリティヘッダーの変更後に確認します。`<オリジンのIP>` はサーバーの実際のIPです。

| 確認内容 | 期待する結果 |
|---|---|
| Cloudflare経由のアクセス | `/healthz` が 204 |
| Cloudflare経由の応答ヘッダー | 手順6の5つのヘッダーが、それぞれ1つずつ付いている |
| nginxのアクセスログの接続元 | CloudflareのIPではなく利用者のIP |
| オリジンの80番と443番へ直接接続 | 接続できない |
| サーバーの外からMySQLのポートへ接続 | 接続できない |
| 新しいSSHセッション | 接続できる |
| サーバーの再起動後 | 上記がすべて同じ結果 |

```bash
curl -sS -o /dev/null -w '%{http_code}\n' https://api.example.com/healthz
```

セキュリティヘッダーを確認します。成功応答とエラー応答の両方で、手順6の5つのヘッダーが1行ずつ表示されることを確認してください。

```bash
curl -sS -o /dev/null -D - https://api.example.com/healthz | grep -iE '^(strict-transport-security|x-content-type-options|x-frame-options|content-security-policy|referrer-policy):'
```

```bash
curl -sS -o /dev/null -D - https://api.example.com/not-found | grep -iE '^(strict-transport-security|x-content-type-options|x-frame-options|content-security-policy|referrer-policy):'
```

自分のIPが記録されることを確認します。nginxのreload直後は、古いワーカーが処理したリクエストが混ざることがあります。

```bash
sudo tail -n 20 /var/log/nginx/access.log | awk '{print $1}'
```

Cloudflareを迂回した接続が拒否されることを、サーバーの外から確認します。

```bash
curl -k -m 10 --resolve api.example.com:443:<オリジンのIP> https://api.example.com/healthz
```

Cloudflare Workersからのアクセスは、接続元がCloudflareの範囲（`2a06:98c0::/29`）のIPとして記録されることがあります。これは異常ではありません。

## 注意点

- 実クライアントIPを復元すると、nginxとアプリケーションのログに利用者のIPが記録されます。ログの保持期間とアクセス権限を確認してください。
- オリジンへ直接接続する外形監視や外部サービスがあると、手順4で接続できなくなります。Cloudflare経由のURLへ切り替えてください。
