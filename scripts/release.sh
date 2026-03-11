#!/bin/bash
# JPY CLI 发布脚本
# 用法: ./scripts/release.sh [版本号]
# 示例: ./scripts/release.sh v1.0.0

set -e

VERSION=${1:-$(date +v%Y%m%d)}
DIST_DIR="dist"
RELEASE_DIR="release"

echo "=========================================="
echo "  JPY CLI 发布脚本"
echo "  版本: $VERSION"
echo "=========================================="

# 清理
echo "[1/5] 清理旧文件..."
rm -rf $DIST_DIR $RELEASE_DIR
mkdir -p $DIST_DIR $RELEASE_DIR

# 编译
echo "[2/5] 交叉编译..."
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output="$DIST_DIR/jpy-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        output="${output}.exe"
    fi

    echo "  编译 $GOOS/$GOARCH..."
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.Version=$VERSION" -o $output ./cmd/jpy-cli
done

# 打包
echo "[3/5] 打包压缩..."
cd $DIST_DIR

for file in jpy-*; do
    if [[ "$file" == *.exe ]]; then
        name="${file%.exe}"
        zip -q "../$RELEASE_DIR/${name}.zip" "$file"
        echo "  创建 ${name}.zip"
    else
        tar -czf "../$RELEASE_DIR/${file}.tar.gz" "$file"
        echo "  创建 ${file}.tar.gz"
    fi
done

cd ..

# 生成校验和
echo "[4/5] 生成校验和..."
cd $RELEASE_DIR
shasum -a 256 * > checksums.txt
cd ..

echo "[5/5] 完成!"
echo ""
echo "发布文件位于: $RELEASE_DIR/"
ls -lh $RELEASE_DIR/
echo ""
echo "下一步:"
echo "  1. git add . && git commit -m 'release: $VERSION'"
echo "  2. git tag $VERSION"
echo "  3. git push && git push --tags"
echo "  4. gh release create $VERSION $RELEASE_DIR/* --title '$VERSION' --notes '发布说明'"
