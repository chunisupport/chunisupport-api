# logrotate設定手順

Linux 環境で `chunisupport-api` のアプリログとアクセスログを `logrotate` で管理する手順です。

## 前提

設定ファイルでは固定ファイル名を指定してください。

```json
{
  "logging": {
    "level": "info",
    "app_file": "/var/log/chunisupport-api/app.log",
    "access_file": "/var/log/chunisupport-api/access.log",
    "stdout": false
  }
}
```

`stdout=false` の場合、`app_file` と `access_file` は必須です。

## ログディレクトリ作成

アプリケーション実行ユーザーを `chunisupport` とする例です。

```bash
sudo install -d -o chunisupport -g chunisupport -m 0750 /var/log/chunisupport-api
```

## systemd設定

`SIGHUP` でログファイルを開き直せるように、ユニットへ `ExecReload` を追加します。

```ini
[Service]
ExecReload=/bin/kill -HUP $MAINPID
```

設定変更後に systemd を再読み込みします。

```bash
sudo systemctl daemon-reload
```

## logrotate設定

`/etc/logrotate.d/chunisupport-api` を作成します。

```conf
/var/log/chunisupport-api/*.log {
    daily
    rotate 14
    missingok
    notifempty
    compress
    delaycompress
    dateext
    create 0640 chunisupport chunisupport
    sharedscripts
    postrotate
        systemctl reload chunisupport-api >/dev/null 2>&1 || true
    endscript
}
```

`copytruncate` は使用しません。ローテート後に `systemctl reload` でアプリケーションへ `SIGHUP` を送り、アプリケーション側でログファイルを再オープンします。

## 監視

`SIGHUP` による再オープンが失敗した場合、アプリケーションログに `Failed to reopen logs` が出力されます。
このログが出た場合は、`reopen app log` / `reopen access log` のどちらが含まれるかを確認し、対象ファイルの親ディレクトリ、所有者、パーミッションを確認してください。
失敗中は次の再オープン成功まで古いファイルディスクリプタへ書き込み続ける可能性があるため、監視対象に含めてください。

## 確認

設定内容をドライランで確認します。

```bash
sudo logrotate -d /etc/logrotate.d/chunisupport-api
```

強制ローテートで動作確認します。

```bash
sudo logrotate -f /etc/logrotate.d/chunisupport-api
sudo systemctl status chunisupport-api
ls -l /var/log/chunisupport-api
```

`app.log` と `access.log` が再作成され、ローテート後も新しいログが現行ファイルへ出力されていれば完了です。
