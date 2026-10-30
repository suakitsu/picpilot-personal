# PicPilot Personal

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

セルフホスト向けの PicPilot 個人版です。設定画面で画像・テキスト・動画の Provider を登録し、ワークベンチ、制作スタジオ、自由キャンバスから利用できます。業務バックエンドは Go、フロントエンドは React/Next.js です。プラットフォーム課金、ポイント販売、管理者サイトは含みません。

**現在は公開済みの移行検証版であり、本番利用可能との宣言はまだ行っていません。** Java 版の本番データベースには接続しないでください。独立したスキーマを使用し、既存ユーザーや決済記録の自動移行は行いません。

## クイックスタート

Git、起動済みの Docker Engine / Docker Desktop、Docker Compose v2 が必要です。初回ビルドではイメージと依存パッケージをダウンロードします。4 CPU コア、空きメモリ 8 GB、空きディスク 10 GB 以上を推奨します。

```sh
git clone https://github.com/suakitsu/picpilot-personal.git
cd picpilot-personal
```

リポジトリは公開済みです。クローンに GitHub トークンは不要です。

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

## コールドスタート検証

新しい利用者に必要なのはこのリポジトリと Docker だけです。ホストへ Go、Node、PostgreSQL を別途インストールする必要はありません。

```powershell
./scripts/init.ps1                 # 初回のみ。`.env` とランダムな DB パスワードを作成
docker compose up -d --build       # Go API、ユーザー画面、ゲートウェイを構築
docker compose ps                   # db/api/user/gateway が running/healthy になるまで待つ
docker compose exec api /picpilot --setup-token
```

初期化トークンは端末に一度だけ表示されます。`http://localhost:8080` を開き、トークンで個人アカウントを作成します。
ホームまたはログイン画面が表示され、API の Compose ヘルスチェックが通れば起動完了です。Linux/macOS では最初のコマンドを
`sh scripts/init.sh` に置き換えます。初回ビルド後の再起動は `docker compose start` だけで、再ビルドは不要です。

空のデータベースを使った独立検証で、イメージ構築、マイグレーション、最初のアカウント、ログイン、個人 Provider の暗号化保存、
Mock 画像/テキストタスク、冪等性、署名メディア、ログアウトを確認しました。以下はローカルブラウザー検証の画面であり、顧客データや有料モデルは含みません。

以下は以前に完全なデスクトップブラウザー画面から取得した 1440px の検証画像です。モバイル表示や細長い切り抜きではありません。

![個人版ワークベンチ](docs/screenshots/personal-workbench-desktop.png)

![個人版 Provider 設定](docs/screenshots/personal-providers-desktop.png)

### 実 Provider の表示例（2026-10-30）

隔離したローカルデータディレクトリで個人ユーザーが OpenAI Images 互換の Sualoom Provider（`gpt-image-2`）を設定し、個人版の自由創作から 1K 正方形の成人 Lolita ファッション写真を実際に生成しました。下の画像は成功した成品詳細ページの実画面で、上の Mock 検証画像とは分けています。既定のテスト手順ではなく、上流サービスの規約により料金が発生する場合があります。

![個人版 Sualoom gpt-image-2 実 Lolita 結果](docs/screenshots/personal-sualoom-real-lolita.jpg)

トラブルシューティング：

- `POSTGRES_PASSWORD` がない場合は `init.ps1`/`init.sh` を実行します。本物の認証情報を `.env.example` に書かないでください。
- Docker Engine が利用できない場合は Docker Desktop を起動し、`docker info` が Server 情報を返すことを確認します。
- ポート競合時は `.env` の `PICPILOT_HTTP_PORT` と `PICPILOT_PUBLIC_URL` を一緒に変更します。
- ページが開かない場合は `docker compose ps` と `docker compose logs --tail=100 api gateway` を確認し、先にボリュームを削除しないでください。
- 通常停止は `docker compose stop`、再開は `docker compose start` です。`down -v` は DB とアプリデータを削除します。[運用手順](docs/operations.md) にバックアップ、復元、更新、ロールバックを記載しています。

## Provider とローカルゲートウェイ

画像は OpenAI 画像 API と Gemini、テキストは OpenAI Chat Completions / Responses、動画は APIMart・HFSY・Agnes の設定済みアダプターに対応します。実際の API と互換性のあるプロトコルを選択してください。サービス名を変更するだけでは互換性は得られません。

既定の localhost `PICPILOT_PUBLIC_URL` では、`http://127.0.0.1:8888/v1` のような信頼したローカルゲートウェイを自動的に許可します。コンテナ内の `127.0.0.1` はホスト PC ではなくそのコンテナを指すため、Docker Desktop では `host.docker.internal` を使用してください。公開環境では `PICPILOT_ALLOW_PRIVATE_PROVIDERS=false` を設定し、認証のないローカルゲートウェイを公開しないでください。詳細は [設定](docs/configuration.md) を参照してください。

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

## 謝辞

キャンバス機能を提供してくださった以下のオープンソースプロジェクトに感謝します。個人版はライセンス、対応ソース、由来の記録を保持し、PicPilot のブランド、認証、タスク連携、個人保存機能を追加しています。

- [Nova Image Studio](https://github.com/tianjiangqiji/nova-image-studio)：制作スタジオ。バージョン `3.1.3`、固定コミット `7768f3f8d7f47e04c6d18572837a086c7a533161`、AGPL-3.0。
- [Infinite Canvas](https://github.com/basketikun/infinite-canvas)：自由キャンバス。固定コミット `705fdd9b2f7bfe6fd58156fd5c1cdb1ad251c767`、GNU AGPL v3。

ライセンス、由来、変更内容は `third_party/nova-image-studio/UPSTREAM.md` と `third_party/infinite-canvas/PROVENANCE.txt` にあります。

## エディションとライセンス

複数ユーザー運営、決済、管理者機能が必要な場合は、独立した `suakitsu/picpilot-platform` を使用してください。元の Java 版は変更せず、独立して維持します。

PicPilot の suakitsu が権利を持つ独自コードには [MIT](LICENSE) を適用します。同梱の第三者製キャンバスと `core/nova/defaults/` の Nova データは AGPL-3.0 を維持します。これらを含むアプリケーション全体は AGPL-3.0 で配布され、MIT のみのプロジェクトではありません。適用範囲、第三者ライセンス、対応ソースの義務は[ライセンス説明](LICENSING.md)を参照してください。
