# smart-cd 設計書

**ドキュメントID:** SCD-DES-001  
**バージョン:** 1.0.0  
**作成日:** 2026-03-07  
**ステータス:** Draft  
**対応要件定義書:** SCD-REQ-001 v1.0.0

---

## 1. システム概要

### 1.1 構成概要

`cd` はシェルビルトインコマンドであるため、外部プロセスから直接カレントディレクトリを変更することはできない。そのため smart-cd は以下の 2 層構成で実現する。

```
┌──────────────────────────────────────────────────┐
│  Shell Layer (bash / zsh)                        │
│                                                  │
│  function cd() {                                 │
│    local result                                  │
│    result=$(scd "$@" 2>/tmp/scd-err)             │
│    local exit_code=$?                            │
│    cat /tmp/scd-err >&2   # stderr を転送        │
│    [ $exit_code -eq 0 ] && builtin cd "$result"  │
│  }                                               │
└──────────────────────────────────────────────────┘
                      ↕ exec / stdout
┌──────────────────────────────────────────────────┐
│  scd  (Go バイナリ)                              │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  fuzzy   │  │bookmark  │  │   history    │   │
│  │  search  │  │ manager  │  │   manager    │   │
│  └──────────┘  └──────────┘  └──────────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  stack   │  │  config  │  │   selector   │   │
│  │ manager  │  │  loader  │  │  (fzf/内蔵)  │   │
│  └──────────┘  └──────────┘  └──────────────┘   │
│                                                  │
│  stdout: 移動先パス (1行)                        │
│  stderr: UI表示・エラーメッセージ                │
└──────────────────────────────────────────────────┘
                      ↕ file I/O
┌──────────────────────────────────────────────────┐
│  Storage Layer                                   │
│  ~/.config/smart-cd/config.toml                  │
│  ~/.config/smart-cd/bookmarks.json               │
│  ~/.local/share/smart-cd/history.db (SQLite)     │
│  ~/.local/share/smart-cd/stack  (テキスト)       │
└──────────────────────────────────────────────────┘
```

### 1.2 設計方針

- **Go バイナリ完結**: `find` / `sqlite3` 等の外部コマンドに依存しない
- **stdout / stderr の厳密な分離**: シェルラッパーが `$(scd ...)` でパスをキャプチャするため、ユーザー向け出力は必ず stderr に流す
- **フォールバック保証**: `scd` が異常終了した場合、シェルラッパーは `builtin cd` を呼び出さず、エラーメッセージのみ表示する
- **ステートレス設計**: `scd` はリクエスト単位で起動・終了する。状態はすべてファイルに永続化する

---

## 2. ディレクトリ構成

```
smart-cd/
├── cmd/
│   └── scd/
│       └── main.go          # エントリーポイント
├── internal/
│   ├── cli/
│   │   ├── root.go          # cobra ルートコマンド・引数ルーティング
│   │   └── init.go          # --init でシェルスクリプトを出力
│   ├── fuzzy/
│   │   ├── search.go        # ディレクトリ再帰検索・スコアリング
│   │   └── search_test.go
│   ├── bookmark/
│   │   ├── manager.go       # ブックマークCRUD
│   │   └── manager_test.go
│   ├── history/
│   │   ├── manager.go       # 履歴記録・frecency 計算
│   │   ├── db.go            # SQLite アクセス層
│   │   └── manager_test.go
│   ├── stack/
│   │   ├── manager.go       # スタック push/pop/list
│   │   └── manager_test.go
│   ├── selector/
│   │   ├── selector.go      # fzf / 内蔵UI の抽象インターフェース
│   │   ├── fzf.go           # fzf ラッパー
│   │   └── internal.go      # go-fuzzyfinder ラッパー
│   ├── config/
│   │   ├── config.go        # 設定ファイルの読み込み・デフォルト値
│   │   └── config_test.go
│   ├── pathutil/
│   │   ├── pathutil.go      # パス正規化・検証・セキュリティチェック
│   │   └── pathutil_test.go
│   └── output/
│       └── output.go        # カラー出力・エラー出力ヘルパー
├── shell/
│   ├── init.bash            # bash 用ラッパー関数テンプレート
│   └── init.zsh             # zsh 用ラッパー関数テンプレート
├── go.mod
├── go.sum
├── .goreleaser.yaml
└── Makefile
```

---

## 3. パッケージ設計

### 3.1 `cmd/scd/main.go`

エントリーポイント。`cli.Execute()` を呼び出すのみ。

```go
func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

---

### 3.2 `internal/cli` — コマンドルーティング

cobra を使用してサブコマンドを管理する。ただし `cd` の自然なインターフェース（`cd proj`, `cd @name`, `cd -1` 等）を維持するため、引数のプレフィックスで処理を振り分けるルーターを実装する。

#### 引数ルーティングロジック

```
引数なし          → builtin cd 相当 (ホームへ)
@<name>          → bookmark.Jump(name)
-<N> (数値)      → history.JumpN(N)
-H               → history.Interactive()
-a <name>        → bookmark.Add(name)
-d <name>        → bookmark.Delete(name)
-l               → bookmark.List()
-e               → bookmark.Edit()
-g <query>       → fuzzy.SearchGlobal(query)
-p <path>        → stack.Push(path)
--               → stack.Pop()
-s               → stack.List()
--clear-history  → history.Clear()
--config         → config.Edit()
--init <shell>   → cli.PrintInitScript(shell)
--version        → cli.PrintVersion()
--help           → cli.PrintHelp()
<query>          → fuzzy.Search(query)  ← デフォルト
```

#### `--init` によるシェルスクリプト出力

`scd --init bash` / `scd --init zsh` を実行すると、シェル連携用のラッパー関数を stdout に出力する。ユーザーは `eval "$(scd --init bash)"` を RC ファイルに記載するだけでインストールが完了する。

---

### 3.3 `internal/fuzzy` — ファジー検索

#### 検索アルゴリズム

1. `filepath.WalkDir` でディレクトリを再帰走査（深さ上限: `config.MaxDepth`）
2. 除外パターン（glob）に一致するディレクトリはスキップ
3. 各ディレクトリ名に対してスコアを計算し、上位候補を返す

#### スコアリング

| 条件 | スコア加算 |
|------|-----------|
| ディレクトリ名が query と完全一致 | +100 |
| ディレクトリ名が query で始まる | +50 |
| ディレクトリ名に query が含まれる | +20 |
| パスの浅い階層にある（depth が小さい） | +(MaxDepth - depth) × 5 |
| frecency スコアが高い（履歴との照合） | +0〜30 |

候補が 1 件 → 即座に stdout へ出力  
候補が複数 → `selector` に渡してインタラクティブ選択

#### 主要な型・関数

```go
type SearchResult struct {
    Path  string
    Score int
    Depth int
}

func Search(root, query string, cfg *config.Config) ([]SearchResult, error)
func SearchGlobal(query string, cfg *config.Config) ([]SearchResult, error)
```

---

### 3.4 `internal/bookmark` — ブックマーク管理

#### データ構造

```go
type Bookmark struct {
    Name      string    `json:"name"`
    Path      string    `json:"path"`
    CreatedAt time.Time `json:"created_at"`
}

type Store struct {
    Bookmarks []Bookmark `json:"bookmarks"`
}
```

#### ファイル操作

- 読み書きは `bookmarks.json` に対してアトミックに行う（書き込みは一時ファイル → rename）
- ファイルが存在しない場合は空の Store を返す

#### 主要な関数

```go
func Load(path string) (*Store, error)
func (s *Store) Save(path string) error
func (s *Store) Add(name, dirPath string) error
func (s *Store) Delete(name string) error
func (s *Store) Find(name string) (*Bookmark, error)
func (s *Store) List() []Bookmark
```

---

### 3.5 `internal/history` — 履歴管理

#### DB スキーマ

```sql
CREATE TABLE IF NOT EXISTS history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    visit_count INTEGER NOT NULL DEFAULT 1,
    last_visit  TEXT    NOT NULL,
    created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_last_visit ON history(last_visit);
```

#### frecency スコア計算式

訪問から経過した時間に応じて重みを変える。

```
weight(hours) =
    4.0   (1時間以内)
    2.0   (1日以内)
    1.0   (1週間以内)
    0.5   (1ヶ月以内)
    0.25  (それ以降)

frecency_score = visit_count × weight(hours_since_last_visit)
```

#### 主要な関数

```go
func Open(dbPath string) (*DB, error)
func (db *DB) Record(path string) error
func (db *DB) List(sort SortOrder, limit int) ([]Entry, error)
func (db *DB) GetByIndex(n int) (*Entry, error)
func (db *DB) Clear() error
func (db *DB) Prune(maxEntries int) error  // 上限超過時に古いエントリを削除
```

---

### 3.6 `internal/stack` — スタック管理

スタックはセッションをまたいで保持する。`~/.local/share/smart-cd/stack` にパスを1行1エントリで保存する。

> スタックはシェルセッション単位で独立させることも検討したが、シンプルさを優先してプロセス間共有のファイルベースとする。セッション分離は将来拡張で対応する。

#### ファイル形式

```
/home/user/dev/my-project/src
/home/user/dev/other-project
```

#### 主要な関数

```go
func Load(path string) (*Stack, error)
func (s *Stack) Push(dirPath string) error
func (s *Stack) Pop() (string, error)
func (s *Stack) List() []string
func (s *Stack) Save(path string) error
```

---

### 3.7 `internal/selector` — インタラクティブ選択

fzf の有無を起動時に検出し、利用可能であれば fzf を優先する。

```go
type Selector interface {
    Select(candidates []string, prompt string) (string, error)
}

func New(cfg *config.Config) Selector {
    switch cfg.UI.FuzzyFinder {
    case "fzf":
        if which("fzf") {
            return &FzfSelector{}
        }
        return &InternalSelector{}  // フォールバック
    case "peco":
        if which("peco") {
            return &PecoSelector{}
        }
        return &InternalSelector{}
    default:
        return &InternalSelector{}
    }
}
```

#### FzfSelector

`fzf` を子プロセスとして起動し、候補を stdin に渡す。選択結果を stdout から受け取る。TTY は `/dev/tty` を直接開いて使用する（`$(scd ...)` によるパイプ環境でも動作させるため）。

#### InternalSelector

`github.com/ktr0731/go-fuzzyfinder` を使用した純 Go 実装のフォールバック UI。

---

### 3.8 `internal/config` — 設定管理

#### 設定の優先順位（高い順）

1. 環境変数（`SMART_CD_MAX_DEPTH` 等）
2. `~/.config/smart-cd/config.toml`
3. デフォルト値

#### Go 構造体

```go
type Config struct {
    Search  SearchConfig  `toml:"search"`
    History HistoryConfig `toml:"history"`
    UI      UIConfig      `toml:"ui"`
}

type SearchConfig struct {
    MaxDepth        int      `toml:"max_depth"`         // default: 5
    GlobalRoot      string   `toml:"global_root"`       // default: "~"
    ExcludePatterns []string `toml:"exclude_patterns"`  // default: ["node_modules",".git","dist",".cache"]
}

type HistoryConfig struct {
    MaxEntries int    `toml:"max_entries"`  // default: 1000
    Sort       string `toml:"sort"`         // default: "frecency"
}

type UIConfig struct {
    Color       bool   `toml:"color"`        // default: true
    FuzzyFinder string `toml:"fuzzy_finder"` // default: "fzf"
}
```

#### ファイルパスの解決

| データ種別 | パス |
|-----------|------|
| 設定ファイル | `$XDG_CONFIG_HOME/smart-cd/config.toml`（未設定時: `~/.config/smart-cd/config.toml`） |
| ブックマーク | `$XDG_CONFIG_HOME/smart-cd/bookmarks.json` |
| 履歴DB | `$XDG_DATA_HOME/smart-cd/history.db`（未設定時: `~/.local/share/smart-cd/history.db`） |
| スタック | `$XDG_DATA_HOME/smart-cd/stack` |

---

### 3.9 `internal/pathutil` — パス検証

セキュリティ上の理由から、すべてのパス入力をこのパッケージで検証する。

```go
// Resolve はチルダ展開・シンボリックリンク解決・絶対パス化を行う
func Resolve(path string) (string, error)

// IsSafe はパストラバーサル攻撃のパターン（../../ 等）を検出する
func IsSafe(path string) bool

// Exists はパスが実際に存在し、ディレクトリであることを確認する
func Exists(path string) bool
```

---

## 4. シェル連携設計

### 4.1 bash 用ラッパー (`init.bash`)

```bash
function cd() {
    # 引数なしはホームへ（標準動作）
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    # scd を実行し stdout をキャプチャ、stderr はターミナルへ
    local target
    target=$(scd "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        builtin cd "$target"
    fi
    return $exit_code
}

# Tab 補完の登録
_scd_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    # ブックマーク名の補完
    if [[ "$cur" == @* ]]; then
        local names
        names=$(scd --list-bookmarks 2>/dev/null)
        COMPREPLY=($(compgen -W "$names" -- "$cur"))
    fi
}
complete -F _scd_completion cd
```

### 4.2 zsh 用ラッパー (`init.zsh`)

```zsh
function cd() {
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    local target
    target=$(scd "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        builtin cd "$target"
    fi
    return $exit_code
}

# zsh 補完
_scd_complete() {
    if [[ "$words[2]" == @* ]]; then
        local -a bookmarks
        bookmarks=($(scd --list-bookmarks 2>/dev/null))
        compadd -P @ -- "${bookmarks[@]#@}"
    fi
}
compdef _scd_complete cd
```

### 4.3 stdout / stderr の使い分け

| 出力内容 | 出力先 | 理由 |
|---------|--------|------|
| 移動先パス（1行） | stdout | シェルラッパーが `$()` でキャプチャするため |
| fzf / 内蔵UI | stderr (/dev/tty) | インタラクティブUIはターミナルに直接表示する必要があるため |
| エラーメッセージ | stderr | ユーザー向け表示はすべて stderr |
| ブックマーク一覧・履歴一覧 | stderr | 同上 |

---

## 5. データフロー設計

### 5.1 ファジー検索フロー

```
cd proj
  │
  ▼
cli.Route("proj")
  │
  ▼
fuzzy.Search(cwd, "proj", config)
  │  filepath.WalkDir で再帰走査
  │  除外パターンチェック
  │  スコアリング
  ▼
[]SearchResult
  │
  ├─ 0件 → stderr: "not found" → exit 1
  ├─ 1件 → stdout: path → exit 0
  └─ 複数 → selector.Select(candidates)
              │
              ├─ 選択 → history.Record(path)
              │          stdout: path → exit 0
              └─ キャンセル → exit 1
```

### 5.2 ブックマークジャンプフロー

```
cd @myproj
  │
  ▼
cli.Route("@myproj")
  │
  ▼
bookmark.Find("myproj")
  │
  ├─ 見つからない → stderr: "bookmark 'myproj' not found" → exit 1
  └─ 見つかった
       │
       ▼
     pathutil.Exists(path)?
       │
       ├─ No  → stderr: "path no longer exists: ..." → exit 1
       └─ Yes → history.Record(path)
                 stdout: path → exit 0
```

### 5.3 履歴記録フロー

`cd` による移動が成功するたびにシェルラッパーから `scd --record <path>` を呼び出して履歴に記録する。

```bash
# init.bash の cd 関数内（移動成功後）
if builtin cd "$target"; then
    scd --record "$target" &  # バックグラウンドで非同期実行（応答速度に影響させない）
fi
```

---

## 6. エラーハンドリング設計

### 6.1 終了コード

| コード | 意味 |
|--------|------|
| 0 | 成功（stdout にパスを出力済み） |
| 1 | 一般エラー（候補なし・ブックマーク未存在等） |
| 2 | 設定ファイル・DB の読み込みエラー |
| 130 | ユーザーによるキャンセル (Ctrl+C) |

### 6.2 エラーメッセージ方針

- 日本語・英語の切り替えは `LANG` 環境変数に従う（初期実装は英語のみ、i18n は将来対応）
- エラーメッセージには原因と対処法を含める

```
Error: bookmark 'myproj' not found
Hint:  run 'cd -l' to list available bookmarks
```

### 6.3 フォールバック

`scd` が exit 1 以上で終了した場合、シェルラッパーは `builtin cd` を呼び出さない。これにより誤ったパスへの移動を防ぐ。

---

## 7. テスト設計

### 7.1 ユニットテスト方針

各パッケージに `_test.go` を配置し、外部依存（ファイルシステム・DB）は `t.TempDir()` を使ってテスト専用ディレクトリで完結させる。

```go
// 例: history パッケージのテスト
func TestRecord(t *testing.T) {
    db, _ := Open(filepath.Join(t.TempDir(), "history.db"))
    db.Record("/home/user/dev/project")
    entries, _ := db.List(SortFrecency, 10)
    assert.Equal(t, "/home/user/dev/project", entries[0].Path)
}
```

### 7.2 統合テスト方針

`testscript`（`golang.org/x/tools/txtar`）を使い、シェルスクリプトレベルの E2E テストを記述する。

```
# testdata/fuzzy_search.txt
exec scd proj
stdout '/home/user/dev/my-project'
! stderr .
```

### 7.3 ベンチマーク

```go
func BenchmarkSearch(b *testing.B) {
    // 1000ディレクトリの一時ツリーを生成してベンチマーク
    for b.Loop() {
        fuzzy.Search(root, "proj", cfg)
    }
}
```

---

## 8. ビルド・リリース設計

### 8.1 Makefile ターゲット

```makefile
build:      ## バイナリビルド
    go build -o bin/scd ./cmd/scd

test:       ## テスト実行
    go test ./... -race -count=1

bench:      ## ベンチマーク
    go test ./... -bench=. -benchmem

lint:       ## 静的解析
    golangci-lint run

release:    ## goreleaser でリリースビルド
    goreleaser release --clean
```

### 8.2 goreleaser 設定 (`.goreleaser.yaml`)

```yaml
builds:
  - id: scd
    main: ./cmd/scd
    binary: scd
    env: [CGO_ENABLED=0]   # modernc.org/sqlite を使用してCGO不要にする
    goos: [linux, darwin]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "scd_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"
```

> `CGO_ENABLED=0` を実現するため、SQLite ライブラリは `modernc.org/sqlite`（Pure Go 実装）を採用する。

### 8.3 CI (GitHub Actions)

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ["1.26"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "${{ matrix.go }}" }
      - run: make test
      - run: make bench
```

---

## 9. 将来拡張への考慮

| 拡張項目 | 現在の設計での考慮点 |
|---------|-------------------|
| fish shell 対応 | `--init fish` の分岐を `cli/init.go` に追加するだけで対応可能 |
| Windows 対応 | `pathutil` でパス区切り文字を抽象化済み。シェルラッパーのみ追加が必要 |
| セッション分離スタック | `stack` パッケージの保存パスに `$SHLVL` や PID を含めることで対応可能 |
| i18n | `output` パッケージにメッセージカタログを追加することで対応可能 |
| 自動ブックマーク | `fuzzy.Search` の後処理として `project detector` を挿入する設計にする |
