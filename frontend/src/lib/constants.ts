export const COMMANDS = [
  { value: "plan", label: "/plan - 実装計画作成" },
  { value: "research", label: "/research - 外部情報調査" },
  { value: "discuss", label: "/discuss - アイデア深掘り" },
  { value: "fullstack", label: "/fullstack - フルスタック実装" },
  { value: "go", label: "/go - Go バックエンド実装" },
  { value: "nextjs", label: "/nextjs - Next.js フロントエンド実装" },
] as const;

export const DEV_FOLDERS = [
  "実装/実装待ち",
  "実装/完了",
  "検討中",
  "資料",
  "アーカイブ",
] as const;

export const DEFAULT_PROJECT_PATH = "/Users/user/auto-daysupport-cloudrun/";

export const LOCAL_STORAGE_KEY = "ghostrunner_project";

export const PLAN_APPROVAL_KEYWORDS = [
  "承認をお待ち",
  "waiting for approval",
  "Ready for approval",
] as const;
