#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# This file is used to generate genesis block.
# The content should be consistent across all nodes in this chain.

# chain id
chain_id: {chain_id}

# chain maker version
version: {version}

# chain config sequence
sequence: 0

# The blockchain auth type, shoudle be consistent with auth type in node config (e.g., chainmaker.yml)
# The auth type can be permissionedWithCert, permissionedWithKey, public.
# By default it is permissionedWithCert.
# permissionedWithCert: permissioned blockchain, using x.509 certificate to identify members.
# permissionedWithKey: permissioned blockchain, using public key to identify members.
# public: public blockchain, using public key to identify members.
auth_type: "public"

# Crypto settings
crypto:
  # Hash algorithm, can be SHA256, SHA3_256 and SM3
  hash: {hash_type}

# User contract related settings
contract:
  # If the sql support contract is enabled or not.
  # If it is true, storage.statedb_config.provider in chainmaker.yml should be sql.
  enable_sql_support: false

  # If it is true, Only creators are allowed to upgrade contract.
  only_creator_can_upgrade: false

# Virtual machine related settings
vm:
  #0:chainmaker, 1:zxl, 2:ethereum
  addr_type: 2
  # support vm list
  support_list:
    - "wasmer"
    - "gasm"
    - "evm"
    - "dockergo"
    - "wxvm"
  native:
      multisign:
        enable_manual_run: true

# Block proposing related settings
block:
  # To enable this attribute, ensure that the clock of the node is consistent
  # Verify the transaction timestamp or not
  tx_timestamp_verify: true

  # Transaction timeout, in second.
  # if abs(now - tx_timestamp) > tx_timeout, the transaction is invalid.
  tx_timeout: 600

  # Max transaction count in a block.
  block_tx_capacity: 100

  # Max block size, in MB
  block_size: 10

  # The interval of block proposing attempts, in millisecond.
  # should be within the range of [10,10000]
  block_interval: 10

# Core settings
core:
  # Max scheduling time of a block, in second.
  # [0, 60]
  tx_scheduler_timeout: 10

  # Max validating time of a block, in second.
  # [0, 60]
  tx_scheduler_validate_timeout: 10

  # Used for handling txs with sender conflicts efficiently
  enable_sender_group: false

  # Used for dynamic tuning the capacity of tx execution goroutine pool
  enable_conflicts_bit_window: true

  # Consensus message compression related settings
  # consensus_turbo_config:
    # If consensus message compression is enabled or not(solo could not use consensus message turbo).
    # consensus_message_turbo: false

    # Max retry count of fetching transaction in txpool by txid.
    # retry_time: 500

    # Retry interval of fetching transaction in txpool by txid, in ms.
    # retry_interval: 20

# gas account config
account_config:
  # the flag to control if subtracting gas from transaction's origin account when sending tx.
  enable_gas: false
  # Deprecated，the default gas count set for admin account.
  gas_count: 0
  # the minimum gas count to be subtracted from transaction's origin account for invoking tx.
  default_gas: 100
  # the gas price per byte for invoking tx, accurate to 6 digits after the decimal point.
  default_gas_price: 0.1
  # the minimum gas count to be subtracted from transaction's origin account for installing|upgrading tx.
  install_base_gas: 10000
  # the gas price per byte for installing tx, accurate to 6 digits after the decimal point.
  install_gas_price: 0.001

# Consensus settings
consensus:
  # Consensus type: 1-TBFT, 5-DPOS
  type: {consensus_type}

  # Consensus node list start
  nodes:
    - org_id: "{public_org_id}"
      node_id:
        - "{org1_peerid}"
        - "{org2_peerid}"
        - "{org3_peerid}"
        - "{org4_peerid}"
        - "{org5_peerid}"
        - "{org6_peerid}"
        - "{org7_peerid}"
        - "{org8_peerid}"
        - "{org9_peerid}"
        - "{org10_peerid}"
#        - "{org11_peerid}"
#        - "{org12_peerid}"
#        - "{org13_peerid}"
  # Consensus node list end
  # DPOS config start
  dpos_config: # DPoS
    # ERC20 contract config
    - key: erc20.total
      value: "{erc20_total}"
    - key: erc20.owner
      value: "{org1_peeraddr}"
    - key: erc20.decimals
      value: "18"
    - key: erc20.account:DPOS_STAKE
      value: "{erc20_total}"
    # Stake contract config
    - key: stake.minSelfDelegation
      value: "2500000"
    - key: stake.epochValidatorNum
      value: "{epochValidatorNum}"
    - key: stake.epochBlockNum
      value: "10"
    - key: stake.completionUnbondingEpochNum
      value: "1"
    - key: stake.candidate:{org1_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org2_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org3_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org4_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org5_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org6_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org7_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org8_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org9_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org10_peeraddr}
      value: "2500000"
#    - key: stake.candidate:{org11_peeraddr}
#      value: "2500000"
#    - key: stake.candidate:{org12_peeraddr}
#      value: "2500000"
#    - key: stake.candidate:{org13_peeraddr}
#      value: "2500000"
    - key: stake.nodeID:{org1_peeraddr}
      value: "{org1_peerid}"
    - key: stake.nodeID:{org2_peeraddr}
      value: "{org2_peerid}"
    - key: stake.nodeID:{org3_peeraddr}
      value: "{org3_peerid}"
    - key: stake.nodeID:{org4_peeraddr}
      value: "{org4_peerid}"
    - key: stake.nodeID:{org5_peeraddr}
      value: "{org5_peerid}"
    - key: stake.nodeID:{org6_peeraddr}
      value: "{org6_peerid}"
    - key: stake.nodeID:{org7_peeraddr}
      value: "{org7_peerid}"
    - key: stake.nodeID:{org8_peeraddr}
      value: "{org8_peerid}"
    - key: stake.nodeID:{org9_peeraddr}
      value: "{org9_peerid}"
    - key: stake.nodeID:{org10_peeraddr}
      value: "{org10_peerid}"
#    - key: stake.nodeID:{org11_peeraddr}
#      value: "{org11_peerid}"
#    - key: stake.nodeID:{org12_peeraddr}
#      value: "{org12_peerid}"
#    - key: stake.nodeID:{org13_peeraddr}
#      value: "{org13_peerid}"
  # DPOS config end

  # We can specify other consensus config here in key-value format.
  ext_config:
    # - key: aa
    #   value: chain01_ext11

# Trust roots is used to specify the organizations' root certificates in permessionedWithCert mode.
# When in permessionedWithKey mode or public mode, it represents the admin users.
trust_roots:
  - org_id: "public"
    root:
      - "../config/{org_path}/admin/admin1/admin1.pem"
      - "../config/{org_path}/admin/admin2/admin2.pem"
      - "../config/{org_path}/admin/admin3/admin3.pem"
      - "../config/{org_path}/admin/admin4/admin4.pem"

