#!/bin/bash
# 把当前分支推送到 GitHub。
# token 只在命令行传入、用完即走（不写入 .git/config，也不落盘）。
# 用法：bash push_to_github.sh <GitHub_PAT> [owner]
set -e
OWNER="${2:-linguo2625469}"
TOKEN="$1"
REPO="workbuddy2api-panel"
if [ -z "$TOKEN" ]; then
  echo "用法: bash $0 <GitHub_PAT> [owner]"
  exit 1
fi
cd /opt/workbuddy2api-panel
BR=$(git branch --show-current)
echo "==> 目标: https://github.com/${OWNER}/${REPO}  分支: ${BR}"

git remote set-url origin "https://github.com/${OWNER}/${REPO}.git"

# 敏感文件二次确认（auths/ data/ config.json 等不应入库）
BAD=$(git ls-files | grep -cE 'auths/|config\.json|^data/|request-logs/' || true)
if [ "$BAD" != "0" ]; then
  echo "!! 检测到敏感文件已被追踪，已中止推送："
  git ls-files | grep -E 'auths/|config\.json|^data/|request-logs/'
  exit 1
fi
echo "==> 敏感文件检查通过（无 auths/ config.json data/）"

# 仓库 local config 里有 insteadOf 把 github.com 重写成 gh-proxy（只读代理，
# 只支持 fetch 不支持 push），push 前临时摘掉、退出时恢复，保留后续 pull 的加速。
PROXY_KEY='url.https://gh-proxy.com/https://github.com/.insteadof'
SAVED=$(git config --local --get "$PROXY_KEY" || true)
if [ -n "$SAVED" ]; then
  echo "==> 临时摘除 insteadOf 代理重写（推完自动恢复）"
  git config --local --unset "$PROXY_KEY"
fi
restore() {
  if [ -n "$SAVED" ]; then
    git config --local "$PROXY_KEY" "$SAVED" || true
    echo "==> 已恢复 insteadOf 代理配置"
  fi
}
trap restore EXIT

git -c credential.helper="!f() { echo username=${OWNER}; echo password=${TOKEN}; }; f" \
    push -u origin "refs/heads/${BR}:refs/heads/${BR}"
echo "==> 推送完成: https://github.com/${OWNER}/${REPO}/tree/${BR}"
