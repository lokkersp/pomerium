#!/usr/bin/env bash

DEPLOYMENT=pomerium-authorize
NS=wn-infra
kn='kubectl -n'
${kn} ${NS} port-forward $(${kn} ${NS} get po -l app.kubernetes.io/name=${DEPLOYMENT} -o jsonpath="{.items[0].metadata.name}") 9999