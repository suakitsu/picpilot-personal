# PicPilot Personal Go

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

セルフホスト向けの PicPilot 個人版です。設定画面で画像・テキスト・動画の Provider を登録し、ワークベンチ、制作スタジオ、自由キャンバスから利用できます。業務バックエンドは Go、フロントエンドは React/Next.js です。プラットフォーム課金、ポイント販売、管理者サイトは含みません。

**現在は非公開の移行検証版であり、本番利用可能との宣言はまだ行っていません。** Java 版の本番データベースには接続しないでください。独立したスキーマを使用し、既存ユーザーや決済記録の自動移行は行いません。

## クイックスタート

Git、起動済みの Docker Engine / Docker Desktop、Docker Compose v2 が必要です。初回ビルドではイメージと依存パッケージをダウンロードします。4 CPU コア、空きメモリ 8 GB、空きディスク 10 GB 以上を推奨します。

```sh
git clone https://github.com/suakitsu/picpilot-personal-go.git
cd picpilot-personal-go
```

非公開期間中は自分の GitHub アクセス権が必要です。クローン URL にトークンを埋め込まないでください。

Windows PowerShell:

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS:

```sh
sh scripts/init.sh
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

1. [PicPilot](http://localhost:8080) を開き、ターミナルに表示された初期化トークンで自分のアカウントを作成します。トークンは共有しないでください。
2. ログイン後、設定のモデル構成を開きます。実際の API に対応するプロトコル、モデル ID、URL、自分の API Key を登録します。
3. 設定を確認してから有効化します。設定確認では有料モデルを呼び出しません。実際の生成では上流サービスの料金が発生する場合があります。

既定アカウント、既定パスワード、有料 API Key はありません。初期化スクリプトがランダムなデータベースパスワードを生成し、アプリが永続的な暗号鍵を生成します。他のユーザーや運営者の認証情報を借用せず、プラットフォームのポイントも消費しません。

本リポジトリには `core` の完全なソースを同梱しています。**プラットフォーム版のクローンや非公開モジュールの取得は不要です。** 公開ユーザー登録と管理者サイトは無効です。Next.js は画面表示と通信転送のみを担い、タスク、業務データ、モデル実行は Go が処理します。

## Provider とローカルゲートウェイ

画像は OpenAI 画像 API と Gemini、テキストは OpenAI Chat Completions / Responses、動画は APIMart・HFSY・Agnes の設定済みアダプターに対応します。実際の API と互換性のあるプロトコルを選択してください。サービス名を変更するだけでは互換性は得られません。

コンテナ内の `127.0.0.1` はホスト PC ではなく、そのコンテナを指します。Docker Desktop では `host.docker.internal` を使用できます。必要な場合だけ `PICPILOT_ALLOW_PRIVATE_PROVIDERS=true` を指定してください。認証のないローカルゲートウェイを公開しないでください。詳細は [設定](docs/configuration.md) を参照してください。

## データ管理

- HTTP は既定で `127.0.0.1:8080` のみに公開し、PostgreSQL のホストポートは公開しません。外部公開には HTTPS を使用してください。
- API Key はサーバー側で暗号化し、ブラウザーや埋め込みキャンバスには返しません。
- `.env`、データベース、アップロードファイル、`instance-secrets.json` をまとめてバックアップしてください。暗号鍵を失うと、保存済みの API Key を復号できません。
- 停止時に `docker compose down -v` を使用しないでください。永続ボリュームが削除されます。

```sh
docker compose stop
docker compose start
docker compose logs --tail=100 api
```

[設定と HTTPS](docs/configuration.md) · [バックアップ・復元・更新](docs/operations.md) · [テスト](docs/testing.md)

## エディションとライセンス

複数ユーザー運営、決済、管理者機能が必要な場合は、独立した `suakitsu/picpilot-platform-go` を使用してください。元の Java 版は変更せず、独立して維持します。

[AGPL-3.0](LICENSE) を採用しています。両キャンバスの上流ライセンス表示と変更後の対応ソースを保持します。`third_party/*/UPSTREAM.md` を確認してください。所有者の承認までは非公開です。認証情報、画像、バックアップ、ユーザーデータを Git に登録しないでください。
