Instructions for use:

1. Copy updateEdgeAdapter.etc.default file into /etc/default, name the file "updateEdgeAdapter"
2. Copy updateEdgeAdapter.etc.initd file into /etc/init.d, name the file "updateEdgeAdapter"
3. From a terminal prompt, execute the following commands:
	3a. chmod 755 /etc/init.d/updateEdgeAdapter
	3b. chown root:root /etc/init.d/updateEdgeAdapter
	3c. update-rc.d updateEdgeAdapter defaults 85

If you wish to start the adapter, rather than reboot, issue the following command from a terminal prompt:

	/etc/init.d/updateEdgeAdapter start