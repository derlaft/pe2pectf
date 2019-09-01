#!/usr/bin/env bash

cd $(dirname $(readlink -f "$0"))

set -ex
(cd .. && go build -mod vendor -v .)

# kill all the shit on exit
trap 'kill $(jobs -p)' EXIT

../pe2pectf \
    -debug \
    -crypto-config=./configs/relay-1.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy= \
    -listen-relay=127.0.0.1:4422 &

sleep 5s

../pe2pectf \
    -debug \
    -crypto-config=./configs/team-1.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy=127.0.0.1:9001 \
    -listen-relay=127.0.0.1:4401 &

../pe2pectf \
    -debug \
    -crypto-config=./configs/team-2.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy=127.0.0.1:9002 \
    -listen-relay=127.0.0.1:4402 &

../pe2pectf \
    -debug \
    -crypto-config=./configs/team-3.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy= \
    -listen-proxy=127.0.0.1:9003 \
    -listen-relay=127.0.0.1:4403 &

cat > /dev/null
