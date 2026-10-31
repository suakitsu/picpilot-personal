# PicPilot Personal

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

画像生成、動画生成、EC 制作、参考画像、制作スタジオ、自由キャンバスに対応する個人セルフホスト向けのオープンソース AI ワークスペースです。自分の画像・テキスト・動画 Provider を接続して使います。プラットフォーム課金、ポイント販売、管理画面はありません。

> **状態：公開移行検証版。** 新しいデータベースでデプロイし、Java 版の本番データベースには接続しないでください。

## クイックスタート

Git、Docker Engine/Docker Desktop、Docker Compose v2 が必要です。初回ビルドは 4 CPU コア、メモリ 8 GB、空きディスク 10 GB 程度を推奨します。

```sh
git clone https://github.com/suakitsu/picpilot-personal.git
cd picpilot-personal
```

Windows PowerShell:

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS は最初の行を `sh scripts/init.sh` に置き換え、残りは同じです。

1. `http://localhost:8080` を開き、端末に表示された一度だけのトークンで個人アカウントを作成します。
2. モデル設定を開き、実際のプロトコル、モデル、URL、API Key を Provider に登録します。
3. 確認して有効化し、ワークベンチ、制作スタジオ、自由キャンバスで作成します。

既定アカウント、パスワード、プラットフォームのキーはありません。個人版は現在のユーザーが設定した Provider だけを使い、Provider キーはサーバー側で暗号化してブラウザーやキャンバスには返しません。

## Provider とローカルゲートウェイ

設定済み Provider の Base URL は `http://` と `https://` の両方に対応し、`http://127.0.0.1:8888/v1` のようなローカルゲートウェイも指定できます。コンテナからホストへ接続する場合、Docker Desktop では `host.docker.internal` を使用します。ユーザーが送信した外部画像 URL は引き続き厳格な安全ダウンロードを通ります。

Provider は上流の実際のプロトコルに対応している必要があります。サービス名だけでは互換性は得られません。設定確認で有料モデルは呼び出さず、生成料金は上流サービスに従います。

## 画面プレビュー

![個人版 Provider 設定](docs/screenshots/personal-providers-desktop.png)

![個人版生成結果の例](docs/screenshots/personal-sualoom-real-lolita.jpg)

## 運用

- 既定では `127.0.0.1:8080` のみ公開し、PostgreSQL のホストポートは公開しません。
- 外部公開時はゲートウェイまたはリバースプロキシで HTTPS を設定し、公開 URL を正しく指定してください。
- `.env`、データベース、アップロードファイル、`instance-secrets.json` をバックアップし、Git には登録しないでください。
- 通常停止は `docker compose stop` を使用します。`docker compose down -v` は永続ボリュームを削除します。

詳細：[設定と HTTPS](docs/configuration.md) · [バックアップ・復元・更新](docs/operations.md) · [テスト](docs/testing.md)

## エディションとライセンス

複数ユーザー運営、決済、管理画面が必要な場合は、別リポジトリの [`picpilot-platform`](https://github.com/suakitsu/picpilot-platform) を使用してください。元の Java 版は独立して維持します。

PicPilot の独自コードは [MIT](LICENSE) です。同梱の Nova データと 2 つのキャンバスは AGPL-3.0 を保持し、組み合わせたアプリケーションは AGPL-3.0 で配布します。範囲と第三者通知は [LICENSING.md](LICENSING.md) を参照してください。
