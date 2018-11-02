# updateEdgeAdapter Adapter

The __updateEdgeAdapter__ adapter provides the ability to upgrade the ClearBlade Edge binary running on a gateway with a TCP connection.

The adapter subscribes to a MQTT topic on the ClearBlade Platform in order to receive upgrade requests.


## MQTT Topic Structure
The __updateEdgeAdapter__ adapter utilizes MQTT messaging to communicate with the ClearBlade Platform. The __updateEdgeAdapter__ adapter will subscribe to a specific topic in order to handle upgrade requests. Additionally, the __updateEdgeAdapter__ adapter will publish messages to MQTT topics in order to communicate the results of requests to client applications (mainly the Edge upgrade portal). The topic structures utilized by the __updateEdgeAdapter__ adapter are as follows:

  * Upgrade edge request: {__TOPIC ROOT__}/{EDGE_ID}/request
  * Upgrade edge response: {__TOPIC ROOT__}/{EDGE_ID}/response
  * Upgrade edge status logs: {__TOPIC ROOT__}/{EDGE_ID}/logs

### MQTT Payloads
The JSON payloads expected by and returned from the __updateEdgeAdapter__ adapter should have the following formats:

#### Upgrade Edge request

The json request should be structured as follows:

{
  "version": "4.2.3"
}

#### Upgrade Edge response

The json response will resemble the following:
	
{
  “success”: true|false,
  “error”: “the error message”,
  "version": "edge_version",
}

#### Upgrade Edge status logs

The json response will resemble the following:
	
{
  "logs": [
    "Update Edge request payload received: {\"version\":\"4.2.3\"}\n",
    "Downloading ClearBlade Edge version 4.2.3\n",
    "ClearBlade Edge version 4.2.3 downloaded from Github\n",
    "Stopping running edge from system.d: systemctl stop edge.service\n"
  ]
}

## ClearBlade Platform Dependencies
The __updateEdgeAdapter__ adapter was constructed to provide the ability to communicate with a _System_ defined in a ClearBlade Platform instance. Therefore, the adapter requires a _System_ to have been created within a ClearBlade Platform instance.

Once a System has been created, artifacts must be defined within the ClearBlade Platform system to allow the adapter to function properly. At a minimum: 

  * A device needs to be created in the Auth --> Devices collection. The device will represent the adapter. The _name_ and _active key_ values specified in the Auth --> Devices collection will be used by the adapter to authenticate to the ClearBlade Platform. 

## Usage

### Executing the adapter

`updateEdgeAdapter -systemKey=<SYSTEM_KEY> -systemSecret=<SYSTEM_SECRET> -platformURL=<PLATFORM_URL> -messagingURL=<MESSAGING_URL> -deviceName=<DEVICE_NAME> -password=<DEVICE_ACTIVE_KEY>  -logLevel=<LOG_LEVEL> -edgeInstallDir=<INSTALL_DIR> -serviceName=<SERVICE_NAME>`

   __*Where*__ 

   __systemKey__
  * REQUIRED
  * The system key of the ClearBLade Platform __System__ the adapter will connect to

   __systemSecret__
  * REQUIRED
  * The system secret of the ClearBLade Platform __System__ the adapter will connect to
   
   __deviceName__
  * The device name the adapter will use to authenticate to the ClearBlade Platform
  * Requires the device to have been defined in the _Auth - Devices_ collection within the ClearBlade Platform __System__
  * OPTIONAL
  * Defaults to __updateEdgeAdapter__
   
   __password__
  * REQUIRED
  * The active key the adapter will use to authenticate to the platform
  * Requires the device to have been defined in the _Auth - Devices_ collection within the ClearBlade Platform __System__
   
   __platformUrl__
  * The url of the ClearBlade Platform instance the adapter will connect to
  * REQUIRED

   __messagingUrl__
  * The MQTT url of the ClearBlade Platform instance the adapter will connect to
  * REQUIRED

   __logLevel__
  * The level of runtime logging the adapter should provide.
  * Available log levels:
    * fatal
    * error
    * warn
    * info
    * debug
  * OPTIONAL
  * Defaults to __info__

   __edgeInstallDir__ 
  * The directory where the edge binary was installed
  * OPTIONAL
  * Defaults to __/usr/bin/clearblade__

   __serviceName__ 
  * The name used when installing ClearBlade Edge into system.d or init.d
  * If system.d was used, __DO NOT__ include _.service_ in the __serviceName__ parameter 
  * OPTIONAL
  * Defaults to __edge__

## Setup
---
The __updateEdgeAdapter__ adapter is dependent upon the ClearBlade Go SDK and its dependent libraries being installed. The __updateEdgeAdapter__ adapter was written in Go and therefore requires Go to be installed (https://golang.org/doc/install).

### Adapter compilation
In order to compile the adapter for execution within mLinux, the following steps need to be performed:

 1. Retrieve the adapter source code  
    * ```git clone git@github.com:ClearBlade/UpdateEdge-Adapter.git```
 2. Navigate to the _updateEdgeAdapter_ directory  
    * ```cd updateEdgeAdapter```
 3. Compile the adapter for the gateway architecture
    * ```GOARCH=arm GOARM=5 GOOS=linux go build```



