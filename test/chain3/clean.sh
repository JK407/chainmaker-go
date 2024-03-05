#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
rm -rf ./data
rm -rf ./log
rm -rf ./bin
result=$(docker ps -aq -f name=ci-chain3)
[ -z "$result" ] && echo "No container found" || docker rm -f "$result"
