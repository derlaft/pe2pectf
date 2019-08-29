#!/usr/bin/env bash

cd $(dirname $(readlink -f "$0"))

set -ex
(cd .. && go build -mod vendor -v .)

# kill all the shit on exit
trap 'kill $(jobs -p)' EXIT

../pe2pectf \
    -crypto-config=./configs/relay-1.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy= \
    -listen-relay=127.0.0.1:4422 &

sleep 5s

../pe2pectf \
    -crypto-config=./configs/team-1.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy=127.0.0.1:9001 \
    -listen-relay=127.0.0.1:4401 &

../pe2pectf \
    -crypto-config=./configs/team-2.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy=127.0.0.1:9002 \
    -listen-relay=127.0.0.1:4402 &

../pe2pectf \
    -crypto-config=./configs/team-3.json \
    -exit-node-config=./configs/exit-node.json \
    -network-config=./configs/nmap.ini \
    -listen-proxy= \
    -listen-proxy=127.0.0.1:9003 \
    -listen-relay=127.0.0.1:4403 &

python2 -m SimpleHTTPServer 4041 &
sleep 5s

pepeHash=$(md5sum pepe.txt | cut -d' ' -f1)

for proxyPort in 9001 9002 9003; do
    for gameHost in 10.0.0.1 10.0.0.2 10.0.0.3; do
        curl -x socks5://127.0.0.1:$proxyPort http://$gameHost:4041/pepe.txt | md5sum | grep -q $pepeHash
        echo "Query for proxyPort=$proxyPort gameHost=$gameHost is OK"
    done
done
