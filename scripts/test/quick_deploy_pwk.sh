#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
## deploy ChainMaker and test

node_count=$1
chain_count=$2
alreadyBuild=$3
enableDocker=$4
if [ ! -d "../../tools/chainmaker-cryptogen" ]; then
  echo "not found chainmaker-go/tools/chainmaker-cryptogen"
  echo "  default port is 11391 12391, you can use "
  echo "              cd chainmaker-go/tools"
  echo "              ln -s ../../chainmaker-cryptogen ."
  echo "              cd chainmaker-cryptogen && make"
  exit 0
fi

CURRENT_PATH=$(pwd)
SCRIPT_PATH=$(dirname "${CURRENT_PATH}")
PROJECT_PATH=$(dirname "${SCRIPT_PATH}")


if  [[ ! -n $node_count ]] ;then
    echo "node cnt is empty"
    echo "  you can use "
    echo "             ./quick_deploy_pwk.sh nodeCount chainCount alreadyBuild enableDockerVm"
    echo "             ./quick_deploy_pwk.sh 4 1 false true"
    exit 1
fi
if  [ ! $node_count -eq 1 ] && [ ! $node_count -eq 4 ] && [ ! $node_count -eq 7 ]&& [ ! $node_count -eq 10 ]&& [ ! $node_count -eq 13 ]&& [ ! $node_count -eq 16 ];then
    echo "node cnt should be 1 or 4 or 7 or 10 or 13"
    exit 1
fi
    
function start_chainmaker() {
  cd $SCRIPT_PATH
  ./cluster_quick_stop.sh clean
  if [ "${alreadyBuild}" != "true" ]; then
    echo -e "\n\n【generate】 certs and config..."
    if [ "${enableDocker}" == "true" ]; then
      echo "【sh】 ./prepare_pwk.sh 4 1 11391 12391 32391 22391 -c 1 -l INFO --hash SHA256 -v true  --vtp=tcp --vlog=INFO"
      ./prepare_pwk.sh 4 1 11391 12391 32391 22391 -c 1 -l INFO --hash SHA256 -v true  --vtp=tcp --vlog=INFO
    else
      echo "【sh】 ./prepare_pwk.sh 4 1 11391 12391 32391 22391 -c 1 -l INFO --hash SHA256 -v false  --vtp=tcp --vlog=INFO"
      ./prepare_pwk.sh 4 1 11391 12391 32391 22391 -c 1 -l INFO --hash SHA256 -v false  --vtp=tcp --vlog=INFO
    fi
#    echo -e "\nINFO\n\n\n" | ./prepare_pwk.sh $node_count $chain_count
    echo -e "\n\n【build】 release..."
    ./build_release.sh
  fi
  echo -e "\n\n【start】 chainmaker..."
  ./cluster_quick_start.sh normal
  echo -e "\n\nstart..."
  sleep 5

  echo "【chainmaker】 process..."
  ps -ef | grep chainmaker
  chainmaker_count=$(ps -ef | grep chainmaker | wc -l)
  if [ $chainmaker_count -lt 4 ]; then
    echo "build error"
    exit
  fi

  # backups *.gz
  cd $PROJECT_PATH/build
  mkdir -p bak
  mv release/*.gz bak/
}

function prepare_cmc() {
  if [ "${alreadyBuild}" != "true" ]; then
    echo "【build】 cmc start..."
    cd $PROJECT_PATH
    make cmc
  fi

  echo "【prepare】 cmc cert and sdk..."
  cd $PROJECT_PATH/bin
  pwd
  rm -rf testdata
  mkdir testdata
  cp $PROJECT_PATH/tools/cmc/testdata/sdk_config_pwk.yml testdata/
  sed -i 's/12301/12391/' testdata/sdk_config_pwk.yml
  cp -r $PROJECT_PATH/build/crypto-config/ testdata/
}

function cmc_test() {
  echo "【cmc】 send tx..."
  cd $PROJECT_PATH/bin
  pwd
  ## create contract
  ./cmc client contract user create \
    --contract-name=fact \
    --runtime-type=WASMER \
    --byte-code-path=../test/wasm/rust-fact-2.0.0.wasm \
    --version=1.0 \
    --sdk-conf-path=./testdata/sdk_config_pwk.yml \
    --admin-org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/admin/admin.key,./testdata/crypto-config/wx-org2.chainmaker.org/admin/admin.key,./testdata/crypto-config/wx-org3.chainmaker.org/admin/admin.key,./testdata/crypto-config/wx-org4.chainmaker.org/admin/admin.key \
    --sync-result=true \
    --params="{}"

  ## invoke tx
  ./cmc client contract user invoke \
    --contract-name=fact \
    --method=save \
    --sdk-conf-path=./testdata/sdk_config_pwk.yml \
    --params="{\"file_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
    --sync-result=true \
    --result-to-string=true

  ## query tx
  ./cmc client contract user get \
    --contract-name=fact \
    --method=find_by_file_hash \
    --sdk-conf-path=./testdata/sdk_config_pwk.yml \
    --params="{\"file_hash\":\"ab3456df5799b87c77e7f88\"}" \
    --result-to-string=true
}

function cat_log() {
  sleep 1
  grep --color=auto "all necessary\|ERROR\|put block" $PROJECT_PATH/build/release/chainmaker-*1*/log/system.log
}

start_chainmaker
prepare_cmc
cmc_test
cat_log