#!/usr/bin/env bash

package_name=wmcli 	
platforms=("darwin/arm64" "linux/amd64" "windows/amd64" "windows/386" "darwin/amd64")

rm -rf build && mkdir -p build

for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	output_name=$package_name'-'$GOOS'-'$GOARCH
	if [ $GOOS = "windows" ]; then
		output_name+='.exe'
	fi	

	env GOOS=$GOOS GOARCH=$GOARCH go build -o build/$output_name ./cmd/wmcli/*.go
	if [ $? -ne 0 ]; then
   		echo 'An error has occurred! Aborting the script execution...'
		exit 1
	fi
done
