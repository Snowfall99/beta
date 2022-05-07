#!/bin/bash

N=$1
key='~/.ssh/dumbo'

for ((i = 0; i < N; i++)); do
{
  pubIP0=$(jq ".nodes[$i].PublicIpAddress" nodes.json)
  pubIP=${pubIP0//\"/}

  ssh -oStrictHostKeyChecking=no -i $key ubuntu@${pubIP} "mkdir themix; mkdir client"

  scp -i $key cmd/main ubuntu@${pubIP}:/home/ubuntu/themix/main
  scp -i $key node$i.config ubuntu@${pubIP}:/home/ubuntu/themix/themix.config
  scp -i $key crypto/priv_sk ubuntu@${pubIP}:/home/ubuntu/themix/priv_sk
  scp -i $key crypto/tbls_sk* ubuntu@${pubIP}:/home/ubuntu/themix/tbls_sk*
  scp -i $key client/client ubuntu@${pubIP}:/home/ubuntu/client/client
} &
done

wait
