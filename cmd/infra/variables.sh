#! bin/bash

if [ -z "$1" ]
  then
    echo "Please provide the config path"
    exit 1
fi
echo "param: $1"
configPath=$1
clusterFilename="servers.yaml"
infraFilename="infra.yaml"
clusterPathFilename="$configPath/$clusterFilename"
infraPathFilename="$configPath/$infraFilename"
kubesprayVersion=$(yq  .kubespray_version $infraPathFilename | jq -r .)
scriptPath=$(yq .script_path $infraPathFilename | jq -r .)
kubesprayPath=$(yq .kubespray_path $infraPathFilename | jq -r .)
kubesprayPackageTag=$(yq .kubespary_package_tag $infraPathFilename | jq -r .)
dirVersion=$(echo $kubesprayVersion | sed 's/v//')
kubesprayDir="kubespray-$dirVersion"
kubesprayDownloadUrl="$kubesprayPath$kubesprayVersion$kubesprayPackageTag"

clusterName=$(yq .cluster_name $clusterPathFilename | jq -r .)

echo "clusterPathFilename: $clusterPathFilename"
echo "infraPathFilename: $infraPathFilename"
echo "scriptPath: $scriptPath"
echo "clusterName: $clusterName"
echo "kubesprayVersion: $kubesprayVersion"
echo "kubesprayDir: $kubesprayDir"
echo "kubesprayDownloadUrl: $kubesprayDownloadUrl"