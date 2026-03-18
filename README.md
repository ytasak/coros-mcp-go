# coros-mcp-go

COROS Training Hub の非公式APIを利用して、フィットネスデータをMCPツールとしてLLMクライアントに公開するMCPサーバー（Go実装）。

> Based on [coolexer/coros-mcp](https://github.com/coolexer/coros-mcp) (Python)

## セットアップ

```bash
go build -o coros-mcp-go .
```

## 使い方

```bash
COROS_EMAIL=xxx COROS_PASSWORD=xxx ./coros-mcp-go
```

### 環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| `COROS_EMAIL` | Yes | COROS Training Hubのメールアドレス |
| `COROS_PASSWORD` | Yes | COROS Training Hubのパスワード |
| `COROS_BASE_URL` | No | APIベースURL（デフォルト: `https://teamapi.coros.com`） |

### Claude Desktop設定例

```json
{
  "mcpServers": {
    "coros": {
      "command": "/path/to/coros-mcp-go",
      "env": {
        "COROS_EMAIL": "your@email.com",
        "COROS_PASSWORD": "yourpassword"
      }
    }
  }
}
```

## 提供ツール

| ツール名 | 説明 |
|----------|------|
| `get_user_info` | ユーザープロフィール情報を取得 |
| `get_workouts` | 期間指定でワークアウト一覧を取得 |
| `get_workout_detail` | ワークアウトの詳細データを取得 |
| `get_workout_file` | ワークアウトファイル(.fit/.tcx/.gpx)のダウンロードURLを取得 |
| `get_recent_runs` | 直近N日間のランニングワークアウトを取得 |
| `get_training_summary` | 期間のトレーニングサマリーを取得 |
| `get_workout_comments` | ワークアウトへのコメントを取得 |
| `get_import_list` | インポートされたワークアウト一覧を取得 |

## 注意事項

- 非公式API（COROS Training Hub）を使用しています。予告なく変更される可能性があります
- ログインすると Training Hub Webアプリのセッションが無効化されます
