#!/bin/bash

N=$1
key='~/.ssh/dumbo'

for ((i = 0; i < N; i++)); do
{
  pubIP0=$(jq ".nodes[$i].PublicIpAddress" nodes.json)
  pubIP=${pubIP0//\"/}

  scp -i $key ubuntu@${subIP}:/home/ubuntu/client/node.log node-$i.log
} &
done

wait
