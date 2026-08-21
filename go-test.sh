set -e

# 通用测试脚本：构建镜像 -> 容器内执行测试（跑完即删容器）
# 项目名、测试命令均必传，例如：./go-test.sh go03-volunteer-activity "go test ./..."
[ -n "$1" ] && [ -n "$2" ] || { echo "用法: $0 <项目名> <测试命令>"; exit 1; }
PROJECT_NAME="$1"
TEST_CMD="$2"

./build_benzhi_docker.sh "$PROJECT_NAME"      # 构建镜像（默认 linux/amd64，另一架构见上一节）

# Git Bash 调用 docker.exe 时会把 ./... 当成路径改写。关闭转换，并把命令作为容器 argv
# 而不是 sh -c 的单个字符串，避免空格把命令拆成只执行 `go`。
export MSYS_NO_PATHCONV=1
export MSYS2_ARG_CONV_EXCL='*'
# shellcheck disable=SC2086
if docker run --rm "${PROJECT_NAME}:latest" $TEST_CMD; then
    echo "✅ 测试全部通过"
else
    echo "❌ 测试存在失败"
    exit 1
fi
