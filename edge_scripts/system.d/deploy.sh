#!/bin/bash

#Copy binary to /usr/bin
cp updateEdgeAdapter /usr/bin

#Ensure binary is executable
chmod +x /usr/bin/updateEdgeAdapter

#Set up system.d resources so that updateEdgeAdapter is started when the gateway starts
cp updateEdgeAdapter.service /etc/systemd/system/updateEdgeAdapter.service

#Ensure system.d script is executable
chmod +x /etc/systemd/system/updateEdgeAdapter.service

#Enable the adapter in system.d
systemctl enable updateEdgeAdapter.service

echo "updateEdgeAdapter Deployed"