#!/bin/bash

#Remove updateEdgeAdapter from monit
sed -i '/updateEdgeAdapter.pid/{N;N;N;N;d}' /etc/monitrc

#Remove the init.d script
rm /etc/init.d/updateEdgeAdapter

#Remove the default variables file
rm /etc/default/updateEdgeAdapter

#Remove the binary
rm /usr/bin/updateEdgeAdapter

#restart monit
/etc/init.d/monit restart

