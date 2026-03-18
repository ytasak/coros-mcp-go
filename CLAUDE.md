# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

COROS Training Hubの非公式APIを利用して、フィットネスデータをMCPツールとしてLLMクライアントに公開するMCPサーバー（Go実装）。stdioトランスポートで動作する。

## ビルド・実行

```bash
# ビルド
go build -o coros-mcp-go .

# 実行（環境変数必須）
COROS_EMAIL=xxx COROS_PASSWORD=xxx ./coros-mcp-go

# テスト（現時点ではテストファイルなし）
go test ./...

# フォーマット
gofmt -w .
```

## アーキテクチャ

単一パッケージ（`main`）構成。全ファイルがルートディレクトリに配置。

- **main.go** — MCPサーバーの初期化、全8ツールの登録、`withAuth`による認証ラッパー。ツール引数はstructの`jsonschema`タグでSDKに自動推論させる。
- **auth.go** — セッション管理（`session`構造体、`sync.RWMutex`で排他制御）。`ensureLoggedIn`でログインし、トークンをメモリ保持。
- **api.go** — COROS APIへのHTTPリクエスト（`getJSON`/`postJSON`ヘルパー）と各ツールのビジネスロジック。共通レスポンスは`apiResponse`エンベロープ（`result`が`"0000"`で成功）。
- **constants.go** — ベースURL（`COROS_BASE_URL`環境変数でオーバーライド可）、ワークアウト種別マッピング、認証エラーコード。
- **formatters.go** — 日付・ペース・時間のフォーマットユーティリティ。

## 重要な設計パターン

- **認証フロー**: `withAuth`が全APIコールをラップ。初回は`ensureLoggedIn(force=false)`、認証エラー検出時に`force=true`でリトライ。
- **MCP SDK**: `github.com/modelcontextprotocol/go-sdk`の`mcp.AddTool`で型安全にツール登録。引数structの`jsonschema`タグからスキーマを自動生成。
- **APIレスポンス判定**: `apiResponse.Result == "0000"`が成功。`authErrorCodes`（1003/1004/1005）はトークン期限切れ。
