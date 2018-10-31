#!/bin/bash

#Remove updateEdgeAdapter from init.d
update-rc.d -f updateEdgeAdapter remove

#Remove the init.d script
rm /etc/init.d/updateEdgeAdapter

#Remove the default variables file
rm /etc/default/updateEdgeAdapter

#Remove the binary
rm /usr/bin/updateEdgeAdapter

