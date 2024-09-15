#!/bin/bash

# Array of platforms to build for, including different operating systems and architectures
platforms=("windows/amd64" "windows/386" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")
# Name of the output binary file
output_name="caching-proxy"

# Loop through each platform specified in the array
for platform in "${platforms[@]}"
do
    # Split the platform string into GOOS and GOARCH
    # shellcheck disable=SC2206
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    # Define the directory and file path for the current platform
    output_dir="./release/$GOOS-$GOARCH"
    output_file="$output_dir/$output_name"

    # Append .exe extension for Windows builds
    if [ "$GOOS" = "windows" ]; then
        output_file+='.exe'
    fi

    # Create the output directory for the current platform, if it doesn't exist
    mkdir -p $output_dir
    echo "Building for $GOOS/$GOARCH..."

    # Build the Go application for the specified GOOS and GOARCH
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o $output_file ./cmd/main.go

    # Check if the build command was successful
    if [ $? -ne 0 ]; then
        echo "Error building for $GOOS/$GOARCH"
        exit 1
    fi

    # Package the binary file into a tar.gz archive
    # -C changes to the output directory and packs the file without including directory structure
    tar -czvf "./release/$output_name-$GOOS-$GOARCH.tar.gz" -C "$output_dir" "$(basename $output_file)"

    # Package the binary file into a zip archive
    # -j option for zip to not include the directory structure in the zip file
    zip -j "./release/$output_name-$GOOS-$GOARCH.zip" "$output_file"

    # Remove the output directory to clean up if the directory itself is not needed
    rm -rf "$output_dir"
done
