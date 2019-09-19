#!/usr/bin/env bash



function buildPlugin() {

    PluginName="$1"

    if [[ "$PluginName" != "" ]]; then
        cd  $(dirname $0)
        source="$(pwd)"

        time=$(date "+%Y%m%d%H%M%S")
        tmpDir="${GOPATH}/src/goku-plugin-${time}/"

        mkdir -p $tmpDir
        cp  -r $source/* $tmpDir/
        cd $tmpDir/

        for file in *.go
            do

                 sed -i '1c package main'   $file


            done
        echo -e "module goku-plugin-${time}\n\ngo 1.12\n\nrequire github.com/eolinker/goku-plugin v0.1.3" > go.mod
        go mod vendor

        rm -f ${source}/$PluginName.so
        go build --buildmode=plugin -o ${source}/$PluginName.so
        rm -rf $tmpDir
        cd ${source}
    fi

}

PROJECT_NAME=$(pwd)
buildPlugin "${PROJECT_NAME##*/}"
