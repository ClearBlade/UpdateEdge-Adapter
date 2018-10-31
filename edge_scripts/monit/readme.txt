Instructions for use:

1. Copy updateEdgeAdapter.etc.default file into /etc/default, name the file "updateEdgeAdapter"
2. Copy updateEdgeAdapter.etc.initd file into /etc/init.d, name the file "updateEdgeAdapter"
3. From a terminal prompt, execute the following commands:
	3a. chmod 755 /etc/init.d/updateEdgeAdapter
	3b. chown root:root /etc/init.d/updateEdgeAdapter
	3c. sed -i '/updateEdgeAdapter.pid/{N;N;N;N;d}' /etc/monitrc
	3d. sed -i '/#  check process monit with pidfile/i \
           check process updateEdgeAdapter with pidfile \/var\/run\/updateEdgeAdapter.pid \
           start program = "\/etc\/init.d\/updateEdgeAdapter start" with timeout 60 seconds \
           stop program  = "\/etc\/init.d\/updateEdgeAdapter stop"' /etc/monitrc
	3e. /etc/init.d/monit restart

If you wish to start the adapter, rather than reboot, issue the following command from a terminal prompt:

	monit start updateEdgeAdapter