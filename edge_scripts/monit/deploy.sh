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

#Remove updateEdgeAdapter from monit in case it was already there
sed -i '/updateEdgeAdapter.pid/{N;N;N;d}' /etc/monitrc

#Add the adapter to monit
sed -i '/#  check process apache with pidfile/i \
  check process updateEdgeAdapter with pidfile \/var\/run\/updateEdgeAdapter.pid \
    start program = "\/etc\/init.d\/updateEdgeAdapter start" with timeout 60 seconds \
    stop program  = "\/etc\/init.d\/updateEdgeAdapter stop"' /etc/monitrc

#reload monit
monit reload

echo "updateEdgeAdapter Deployed"