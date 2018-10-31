#!/bin/bash

#Copy binary to /usr/local/bin
cp updateEdgeAdapter /usr/bin

#Ensure binary is executable
chmod +x /usr/bin/updateEdgeAdapter

#Set up init.d resources so that updateEdgeAdapter is started when the gateway starts
cp updateEdgeAdapter.etc.initd /etc/init.d/updateEdgeAdapter
cp updateEdgeAdapter.etc.default /etc/default/updateEdgeAdapter

#Ensure init.d script is executable
chmod +x /etc/init.d/updateEdgeAdapter

#Add the init.d script
update-rc.d updateEdgeAdapter defaults 85

echo "updateEdgeAdapter Deployed"