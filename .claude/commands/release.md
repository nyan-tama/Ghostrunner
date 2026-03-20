
# /release

**ultrathink**

staging ブランチを main にマージし、本番環境にリリースします。

## 計測: 開始

最初に以下のコマンドを実行して開始時刻を記録する:

```bash
date +%s > /tmp/claude-timer-release-start
```

## 前提条件

- staging 環境で動作確認が完了していること
- `/stage` で staging への push が完了していること

## Step 1: ステージング確認

ユーザーにステージング環境での確認状況を聞く:

- ステージング環境で動作確認は完了しましたか？
- 問題は見つかりませんでしたか？

確認が取れたら次に進む。

## Step 2: 本番リリース

`release-manager` エージェントを使用して以下を実行する:

1. staging を main にマージ
2. git push origin main
3. Cloud Build デプロイ確認（トリガー設定済みの場合）
4. staging を main にリセット
5. feat ブランチの削除

## Step 3: 仕様書の移動（必須）

**重要: このステップを絶対にスキップしないこと**

仕様書ファイルがある場合、以下のコマンドを実行して `開発/実装/完了/` に移動する:

```bash
mkdir -p "開発/実装/完了"
mv "開発/実装/実装待ち/<仕様書ファイル名>" "開発/実装/完了/<仕様書ファイル名>"
git add "開発/実装/"
git commit -m "docs: 実装完了した仕様書を完了フォルダに移動"
git push origin main
```

## 実行後

リリース完了後、以下を表示する:

```
本番リリースが完了しました。

Cloud Build トリガーが設定されている場合、本番環境に自動デプロイされます。
本番環境で動作確認を行ってください。
問題が発覚した場合は `/hotfix` で緊急修正を行ってください。
```

## 計測: 終了

全ステップ完了後に以下のコマンドを実行して所要時間を表示する:

```bash
start=$(cat /tmp/claude-timer-release-start) && end=$(date +%s) && elapsed=$((end - start)) && minutes=$((elapsed / 60)) && seconds=$((elapsed % 60)) && echo "/release 所要時間: ${minutes}分${seconds}秒" && echo "$(date +%Y-%m-%d),release,$ARGUMENTS,${elapsed}秒,${minutes}分${seconds}秒" >> ~/ghostrunner-timing.csv
```

## タスク

$ARGUMENTS
