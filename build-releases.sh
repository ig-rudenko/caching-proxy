#!/bin/bash

platforms=("windows/amd64" "windows/386" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")
output_name="caching-proxy"

for platform in "${platforms[@]}"
do
    # shellcheck disable=SC2206
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_dir="./release/$GOOS-$GOARCH"
    output_file="$output_dir/$output_name"

    # Для Windows добавляем расширение .exe
    if [ "$GOOS" = "windows" ]; then
        output_file+='.exe'
    fi

    # Создание директории для текущей платформы
    mkdir -p $output_dir
    echo "Building for $GOOS/$GOARCH..."

    # Сборка бинарного файла
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o $output_file ./cmd/main.go

    if [ $? -ne 0 ]; then
        echo "Error building for $GOOS/$GOARCH"
        exit 1
    fi

    # Упаковка в tar.gz только файла, без директории
    tar -czvf "./release/$output_name-$GOOS-$GOARCH.tar.gz" -C "$output_dir" "$(basename $output_file)"

    # Упаковка в zip только файла, без директории
    zip -j "./release/$output_name-$GOOS-$GOARCH.zip" "$output_file"

    # Очистка директории после упаковки, если не нужны сами папки
    rm -rf "$output_dir"
done
