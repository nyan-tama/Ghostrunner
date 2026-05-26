// Package projects はpatrol_projects.jsonの読み込みを提供する。
//
// # 概要
//
// 巡回対象プロジェクトの設定ファイル（patrol_projects.json）を読み込む共通パッケージ。
// PatrolService（巡回機能）とdashboardパッケージ（ダッシュボード状態集約）で共有して使用する。
//
// # 主要な型
//
//   - Project: 1つの登録プロジェクト（Path, Name）
//   - Config: patrol_projects.jsonのトップレベル構造（Projects配列）
//
// # 主要な関数
//
//   - LoadProjects: JSONファイルからProject一覧を読み込む。
//     ファイルが存在しない場合は(nil, nil)を返し、JSON不正の場合のみエラーを返す。
package projects
