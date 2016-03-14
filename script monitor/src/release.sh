#!/bin/bash -x

if [[ "$#" -ne 1 ]]; then
    echo "Please provide version number"
    exit -1
fi
declare -a DISTRIBUTION_LIST=("darwin amd64" 
                                "darwin 386" 
                                "linux amd64" 
                                "linux 386" 
                                "windows amd64" 
                                "windows 386")

for dist in "${DISTRIBUTION_LIST[@]}"
do
    params=($dist)
    os=${params[0]}
    arch=${params[1]}
    name=""
    ext=""
    if [[ $os == "windows" ]] ; then
        GOOS=$os GOARCH=$arch go build -o oghb-$os-$arch.exe
        zip oghb-$os-$arch-v$1.zip oghb-$os-$arch.exe
        rm oghb-$os-$arch.exe
    else 
        GOOS=$os GOARCH=$arch go build -o oghb-$os-$arch
        tar -czvf oghb-$os-$arch-v$1.tar.gz oghb-$os-$arch
        rm oghb-$os-$arch
    fi
done

mkdir release
mv oghb-* release/
