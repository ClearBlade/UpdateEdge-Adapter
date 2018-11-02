#!/bin/bash

#Stop the updateEdgeAdapter service in system.d
systemctl stop updateEdgeAdapter.service

#Disable the updateEdgeAdapter service in system.d
systemctl disable updateEdgeAdapter.service

#Remove the service file
rm /lib/systemd/system/updateEdgeAdapter.service

#Remove the binary
rm /usr/bin/updateEdgeAdapter

#reload the daemon
systemctl daemon-reload
systemctl reset-failed

