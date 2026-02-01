#!/bin/bash

docker exec $(docker ps -q -f name=gto_nginx) nginx -s reload