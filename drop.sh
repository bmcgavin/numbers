#!/bin/bash

sudo iptables -A INPUT -p tcp --sport 12345 -j DROP
sudo iptables -A OUTPUT -p tcp --dport 12345 -j DROP