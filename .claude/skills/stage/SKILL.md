---
name: stage
description: featブランチをstagingにsquash mergeしデプロイする
disable-model-invocation: true
---


# /stage

**ultrathink**

feat ブランチを staging ブランチに squash merge し、ステージング環境にデプロイします。

## 計測: 開始

最初に以下のコマンドを実行して開始時刻を記録する:

```bash
date +%s > /tmp/claude-timer-stage-start
```

## 前提条件

- feat ブランチ上で作業が完了し、コミット済みであること
- `/coding`、`/go`、`/nextjs` のいずれかで実装・レビュー・テストが完了していること

## 実行方法

`staging-manager` エージェントを使用して以下を実行する:

1. staging ブランチの存在確認（なければ初期化）
2. feat ブランチを staging に squash merge
3. git push origin staging

## 実行後

push 完了後、以下を表示する:

```
staging へのマージ・push が完了しました。

Cloud Build トリガーが設定されている場合、ステージング環境に自動デプロイされます。
ステージング環境で動作確認を行い、問題なければ `/release` で本番リリースしてください。
```

## 計測: 終了

全ステップ完了後に以下のコマンドを実行して所要時間を表示する:

```bash
start=$(cat /tmp/claude-timer-stage-start) && end=$(date +%s) && elapsed=$((end - start)) && minutes=$((elapsed / 60)) && seconds=$((elapsed % 60)) && echo "/stage 所要時間: ${minutes}分${seconds}秒" && echo "$(date +%Y-%m-%d),stage,$ARGUMENTS,${elapsed}秒,${minutes}分${seconds}秒" >> ~/ghostrunner-timing.csv
```

## タスク

$ARGUMENTS
