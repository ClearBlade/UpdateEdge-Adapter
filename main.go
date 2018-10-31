package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cb "github.com/clearblade/Go-SDK"
	mqttTypes "github.com/clearblade/mqtt_parsing"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/logutils"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	msgSubscribeQos    = 0
	msgPublishQos      = 0
	initSysTypeInitd   = "init"
	initSysTypeSystemd = "systemd"
	initSysTypeMonit   = "monit"
	downloadDir        = "/tmp"
)

var (
	platformURL      string //Defaults to http://localhost:9000
	messagingURL     string //Defaults to localhost:1883
	sysKey           string
	sysSec           string
	deviceName       string //Defaults to updateEdgeAdapter
	activeKey        string
	logLevel         string //Defaults to info
	edgeInstallDir   string //Defaults to /usr/bin/clearblade
	serviceName      string
	architecture     string
	initSystem       string
	edgeDownloadName string
	deployLogs       []string
	edgeId           string

	topicRoot                 = "edge/update"
	cbBroker                  cbPlatformBroker
	cbSubscribeChannel        <-chan *mqttTypes.Publish
	endSubscribeWorkerChannel chan string
)

type cbPlatformBroker struct {
	name         string
	clientID     string
	client       *cb.DeviceClient
	platformURL  *string
	messagingURL *string
	systemKey    *string
	systemSecret *string
	username     *string
	password     *string
	topic        string
	qos          int
}

func init() {
	flag.StringVar(&sysKey, "systemKey", "", "system key (required)")
	flag.StringVar(&sysSec, "systemSecret", "", "system secret (required)")
	flag.StringVar(&deviceName, "deviceName", "updateEdgeAdapter", "name of device (optional)")
	flag.StringVar(&activeKey, "password", "", "password (or active key) for device authentication (required)")
	flag.StringVar(&platformURL, "platformURL", "", "platform url (required)")
	flag.StringVar(&messagingURL, "messagingURL", "", "messaging URL (required")
	flag.StringVar(&logLevel, "logLevel", "info", "The level of logging to use. Available levels are 'debug, 'info', 'warn', 'error', 'fatal' (optional)")
	flag.StringVar(&edgeInstallDir, "edgeInstallDir", "/usr/bin/clearblade", "edge installation directory (required)")
	flag.StringVar(&serviceName, "serviceName", "edge", "the name of the init.d or system.d service name Edge is running under (optional)")
}

func usage() {
	log.Printf("Usage: updateEdgeAdapter [options]\n\n")
	flag.PrintDefaults()
}

func validateFlags() {
	flag.Parse()

	if sysKey == "" || sysSec == "" || activeKey == "" || platformURL == "" || messagingURL == "" {

		log.Printf("ERROR - Missing required flags\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	fmt.Println("Starting updateEdgeAdapter...")

	//Validate the command line flags
	flag.Usage = usage
	validateFlags()

	//Initialize the logging mechanism
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		MinLevel: logutils.LogLevel(strings.ToUpper(logLevel)),
		Writer: &lumberjack.Logger{
			Filename:   "/var/log/updateEdgeAdapter",
			MaxSize:    1, // megabytes
			MaxBackups: 5,
			MaxAge:     10, //days
		},
	}
	log.SetOutput(filter)

	cbBroker = cbPlatformBroker{
		name:         "ClearBlade",
		clientID:     deviceName + "_client",
		client:       nil,
		platformURL:  &platformURL,
		messagingURL: &messagingURL,
		systemKey:    &sysKey,
		systemSecret: &sysSec,
		username:     &deviceName,
		password:     &activeKey,
		qos:          msgSubscribeQos,
	}

	//Initialize variables
	initializeVariables()
	if architecture == "" {
		log.Println("Unable to retrieve system architecture. Exiting.")
		os.Exit(-1)
	}

	if edgeId == "" {
		log.Println("Unable to retrieve edge ID, edge is not running. Exiting.")
		os.Exit(-1)
	}

	if edgeDownloadName == "" {
		log.Println("Unable to determine edge binary file name. Exiting.")
		os.Exit(-1)
	}

	// Initialize ClearBlade Client
	var err error
	if err = initCbClient(cbBroker); err != nil {
		log.Println(err.Error())
		log.Println("Unable to initialize CB broker client. Exiting.")
		os.Exit(-1)
	}

	defer close(endSubscribeWorkerChannel)
	endSubscribeWorkerChannel = make(chan string)

	//Handle OS interrupts to shut down gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c

	log.Printf("[INFO] OS signal %s received, ending go routines.", sig)

	//End the existing goRoutines
	endSubscribeWorkerChannel <- "Stop Channel"
	os.Exit(0)
}

func initializeVariables() {
	architecture = getArchitecture()
	initSystem = getInitSystem()
	edgeId = getEdgeId()

	switch architecture {

	case "armv5tejl":
		edgeDownloadName = "edge-linux-armv5tejl.tar.gz"
	case "armv6l":
		edgeDownloadName = "edge-linux-armv6.tar.gz"
	case "armv7l":
		edgeDownloadName = "edge-linux-armv7.tar.gz"
	case "armv8":
		edgeDownloadName = "edge-linux-arm64.tar.gz"
	case "i386", "i686":
		edgeDownloadName = "edge-linux-386.tar.gz"
	case "x86_64":
		edgeDownloadName = "edge-linux-amd64.tar.gz"
	default:
		log.Println("[ERROR] Architecture not supported")
	}
}

// ClearBlade Client init helper
func initCbClient(platformBroker cbPlatformBroker) error {
	log.Println("[DEBUG] initCbClient - Initializing the ClearBlade client")

	cbBroker.client = cb.NewDeviceClientWithAddrs(*(platformBroker.platformURL), *(platformBroker.messagingURL), *(platformBroker.systemKey), *(platformBroker.systemSecret), *(platformBroker.username), *(platformBroker.password))

	for err := cbBroker.client.Authenticate(); err != nil; {
		log.Printf("[ERROR] initCbClient - Error authenticating %s: %s\n", platformBroker.name, err.Error())
		log.Println("[ERROR] initCbClient - Will retry in 1 minute...")

		// sleep 1 minute
		time.Sleep(time.Duration(time.Minute * 1))
		err = cbBroker.client.Authenticate()
	}

	log.Println("[DEBUG] initCbClient - Initializing MQTT")
	callbacks := cb.Callbacks{OnConnectionLostCallback: OnConnectLost, OnConnectCallback: OnConnect}
	if err := cbBroker.client.InitializeMQTTWithCallback(platformBroker.clientID, "", 30, nil, nil, &callbacks); err != nil {
		log.Fatalf("[FATAL] initCbClient - Unable to initialize MQTT connection with %s: %s", platformBroker.name, err.Error())
		return err
	}

	return nil
}

//If the connection to the broker is lost, we need to reconnect and
//re-establish all of the subscriptions
func OnConnectLost(client mqtt.Client, connerr error) {
	log.Printf("[INFO] OnConnectLost - Connection to broker was lost: %s\n", connerr.Error())

	//End the existing goRoutines
	endSubscribeWorkerChannel <- "Stop Channel"

	//We don't need to worry about manally re-initializing the mqtt client. The auto reconnect logic will
	//automatically try and reconnect. The reconnect interval could be as much as 20 minutes.
}

//When the connection to the broker is complete, set up the subscriptions
func OnConnect(client mqtt.Client) {
	log.Println("[INFO] OnConnect - Connected to ClearBlade Platform MQTT broker")

	//CleanSession, by default, is set to true. This results in non-durable subscriptions.
	//We therefore need to re-subscribe
	log.Println("[DEBUG] OnConnect - Begin Configuring Subscription(s)")

	var err error
	for cbSubscribeChannel, err = subscribe(topicRoot + "/" + edgeId + "/request"); err != nil; {
		//Wait 30 seconds and retry
		log.Printf("[ERROR] OnConnect - Error subscribing to MQTT: %s\n", err.Error())
		log.Println("[ERROR] OnConnect - Will retry in 30 seconds...")
		time.Sleep(time.Duration(30 * time.Second))
		cbSubscribeChannel, err = subscribe(topicRoot + "/request/#")
	}

	//Start subscribe worker
	go subscribeWorker()
}

func subscribeWorker() {
	log.Println("[DEBUG] subscribeWorker - Starting subscribeWorker")

	//Wait for subscriptions to be received
	for {
		select {
		case message, ok := <-cbSubscribeChannel:
			if ok {
				handleRequest(message.Payload)
			}
		case _ = <-endSubscribeWorkerChannel:
			//End the current go routine when the stop signal is received
			log.Println("[INFO] subscribeWorker - Stopping subscribeWorker")
			return
		}
	}
}

func handleRequest(payload []byte) {
	log.Printf("[DEBUG] handleRequest - Json payload received: %s\n", string(payload))

	deployLogs = make([]string, 0)

	go deployEdge(payload)
}

func getEdgeId() string {
	cmd := "ps -C edge -f | sed -n 's/.*-edge-id=\\([^ ]*\\)[ ].*/\\1/p'"
	edgeID, err := executeOSCommand("bash", []string{"-c", cmd})
	if err == nil {
		log.Println("[DEBUG] getEdgeId - edge ID is " + edgeID.(string))
		return strings.Replace(edgeID.(string), "\n", "", -1)
	}
	log.Printf("[ERROR] getArchitecture - ERROR retrieving system architecture: %s\n", err.Error())
	return ""
}

func getArchitecture() string {
	arch, err := executeOSCommand("uname", []string{"-m"})
	if err == nil {
		log.Printf("[DEBUG] getArchitecture - architecture is %s\n", strings.Replace(arch.(string), "\n", "", -1))
		return strings.Replace(arch.(string), "\n", "", -1)
	}
	log.Printf("[ERROR] getArchitecture - ERROR retrieving system architecture: %s\n", err.Error())
	return ""
}

func getInitSystem() string {
	//Check if monit is being used
	if isUsingMonit() {
		return initSysTypeMonit
	}

	//May not be foolproof, but will work for now
	log.Println("[DEBUG] getInitSystem - Executing command: ps -p 1")
	psOutput, err := executeOSCommand("ps", []string{"-p", "1"})
	if err == nil {
		if strings.Contains(psOutput.(string), initSysTypeInitd) {
			if isUsingInitd() {
				log.Println("[DEBUG] getInitSystem - init system is init.d")
				return initSysTypeInitd
			}
		} else if strings.Contains(psOutput.(string), initSysTypeSystemd) {
			if isUsingSystemd() {
				log.Println("[DEBUG] getInitSystem - init system is system.d")
				return initSysTypeSystemd
			}
		}
		return ""
	}

	log.Printf("[ERROR] getInitSystem - ERROR retrieving init system: %s\n", err.Error())
	return ""
}

func isUsingInitd() bool {
	log.Printf("[DEBUG] isUsingInitd - Executing command: find /etc/init.d -name %s\n", serviceName)
	findOutput, err := executeOSCommand("find", []string{"/etc/init.d", "-name", serviceName})
	if err == nil {
		if strings.Contains(findOutput.(string), serviceName) {
			log.Printf("[DEBUG] isUsingInitd - '%s' file found in /etc/init.d", serviceName)
			return true
		}
		log.Printf("[DEBUG] isUsingInitd - '%s' file not found in /etc/init.d", serviceName)
		return false
	}
	log.Printf("[ERROR] isUsingInitd - ERROR issuing find command: %s\n", err.Error())
	return false
}

func isUsingSystemd() bool {
	log.Printf("[DEBUG] isUsingSystemd - Executing command: find /lib/systemd/system /etc/systemd/system -name %s.service", serviceName)
	findOutput, err := executeOSCommand("find", []string{"/lib/systemd/system", "/etc/systemd/system", "-name", serviceName + ".service"})
	if err == nil {
		if strings.Contains(findOutput.(string), serviceName+".service") {
			log.Printf("[DEBUG] isUsingSystemd - '%s.service' file found in /lib/systemd/system or /etc/systemd/system", serviceName)
			return true
		}
		log.Printf("[DEBUG] isUsingSystemd - '%s.service' file not found in /lib/systemd/system or /etc/systemd/system", serviceName)
		return false
	}
	log.Printf("[ERROR] isUsingSystemd - ERROR issuing find command: %s\n", err.Error())
	return false
}

func isUsingMonit() bool {
	log.Println("[DEBUG] isUsingMonit - Executing command: ps -C monit")
	psOutput, err := executeOSCommand("ps", []string{"-C", "monit"})

	//See if monit is running
	if err == nil {
		if strings.Contains(psOutput.(string), "monit") {
			//Monit is running, see if monit is used to control edge
			log.Println("[DEBUG] isUsingMonit - Executing command: monit summary")
			monitSummary, err := executeOSCommand("monit", []string{"summary"})
			if err == nil {
				if strings.Contains(monitSummary.(string), "Process '"+serviceName+"'") {
					log.Println("[DEBUG] isUsingMonit - Monit is monitoring edge")
					return true
				}
				log.Println("[DEBUG] isUsingMonit - Monit is not monitoring edge")
				return false
			}
			log.Printf("[ERROR] isUsingMonit - ERROR invoking 'monit summary' command: %s\n", err.Error())
			return false
		}
		log.Println("[DEBUG] isUsingMonit - Monit is running")
		return false

	}

	log.Printf("[ERROR] isUsingMonit - ERROR invoking 'ps -C' command: %s\n", err.Error())
	return false
}

func deployEdge(payload []byte) {
	var err error
	var jsonPayload map[string]interface{}

	addLogEntry(fmt.Sprintf("Update Edge request payload received: %s\n", payload))

	if err := json.Unmarshal(payload, &jsonPayload); err != nil {
		log.Printf("[ERROR] deployEdge - Error encountered unmarshalling json: %s\n", err.Error())
		addErrorToPayload(jsonPayload, "Error encountered unmarshalling json: "+err.Error())
	} else {
		log.Printf("[DEBUG] deployEdge - Json payload received: %#v\n", jsonPayload)
	}

	if jsonPayload["version"] == nil {
		log.Println("[ERROR] deployEdge - version not specified in incoming payload")
		addErrorToPayload(jsonPayload, "The version attribute is required")
	} else {
		var version = jsonPayload["version"].(string)
		//Download Edge
		log.Printf("[DEBUG] deployEdge - Downloading ClearBlade Edge version %s\n", version)
		if err = downloadEdge(version); err != nil {
			addErrorToPayload(jsonPayload, "Error encountered downloading edge: "+err.Error())
		} else {
			//Stop Edge
			log.Println("[DEBUG] deployEdge - Stopping Edge")
			if err = stopEdge(); err != nil {
				addErrorToPayload(jsonPayload, "Error encountered stopping edge: "+err.Error())
			} else {
				//install Edge
				log.Println("[DEBUG] deployEdge - Installing Edge")
				if err = installEdge(version); err != nil {
					addErrorToPayload(jsonPayload, "Error encountered installing edge: "+err.Error())
				}

				//Start Edge
				log.Println("[DEBUG] deployEdge - Starting Edge")
				if err = startEdge(); err != nil {
					addErrorToPayload(jsonPayload, "Error encountered starting edge: "+err.Error())
				}
			}
		}
	}

	if jsonPayload["error"] == nil {
		jsonPayload["success"] = true
	} else {
		jsonPayload["success"] = false
	}

	publishResponse(jsonPayload)
	return
}

func stopEdge() error {
	var err error

	switch initSystem {
	case initSysTypeInitd:
		log.Printf("[DEBUG] stopEdge - Executing command: /etc/init.d/%s stop\n", serviceName)
		addLogEntry(fmt.Sprintf("Stopping running edge from init.d: /etc/init.d/%s stop\n", serviceName))
		_, err = executeOSCommand("/etc/init.d/"+serviceName, []string{"stop"})
	case initSysTypeSystemd:
		log.Printf("[DEBUG] stopEdge - Executing command: systemctl stop %s.service\n", serviceName)
		addLogEntry(fmt.Sprintf("Stopping running edge from system.d: systemctl stop %s.service\n", serviceName))
		_, err = executeOSCommand("systemctl", []string{"stop", serviceName + ".service"})
	case initSysTypeMonit:
		log.Printf("[DEBUG] stopEdge - Executing command: monit stop %s\n", serviceName)
		addLogEntry(fmt.Sprintf("Stopping running edge from monit: monit stop %s\n", serviceName))
		_, err = executeOSCommand("monit", []string{"stop", serviceName})
	}
	if err != nil {
		log.Printf("[ERROR] stopEdge - ERROR stopping edge: %s\n", err.Error())
		return err
	}
	addLogEntry(fmt.Sprintln("Edge stopped"))
	return nil
}

func startEdge() error {
	var err error

	switch initSystem {
	case initSysTypeInitd:
		log.Printf("[DEBUG] startEdge - Executing command: /etc/init.d/%s start\n", serviceName)
		addLogEntry(fmt.Sprintf("Starting edge from init.d: /etc/init.d/%s start", serviceName))
		_, err = executeOSCommand("/etc/init.d/"+serviceName, []string{"start"})
	case initSysTypeSystemd:
		log.Printf("[DEBUG] startEdge - Executing command: systemctl start %s.service\n", serviceName)
		addLogEntry(fmt.Sprintf("Starting edge from system.d: systemctl start %s.service\n", serviceName))
		_, err = executeOSCommand("systemctl", []string{"start", serviceName + ".service"})
	case initSysTypeMonit:
		log.Printf("[DEBUG] startEdge - Executing command: monit start %s\n", serviceName)
		addLogEntry(fmt.Sprintf("Starting edge from monit: monit start %s\n", serviceName))
		_, err = executeOSCommand("monit", []string{"start", serviceName})
	}
	if err != nil {
		log.Printf("[ERROR] startEdge - ERROR starting edge: %s\n", err.Error())
		return err
	}
	addLogEntry(fmt.Sprintln("Edge started"))
	return nil
}

func downloadEdge(version string) error {
	var err error
	//wget -q -O myEdge.tar.gz --no-check-certificate https://github.com/ClearBlade/Edge/releases/download/4.2.3/edge-linux-armv5tejl.tar.gz
	url := "https://github.com/ClearBlade/Edge/releases/download/" + version + "/" + edgeDownloadName

	addLogEntry(fmt.Sprintf("Downloading ClearBlade Edge version %s\n", version))

	log.Printf("[DEBUG] downloadEdge - Executing command: wget -q -P /tmp/ --no-check-certificate %s\n", url)
	if _, err = executeOSCommand("wget", []string{"-q", "-P", "/tmp/", "--no-check-certificate", url}); err != nil {
		log.Printf("[ERROR] downloadEdge - ERROR downloading edge: %s\n", err.Error())
		return errors.New("Error downloading edge binary: " + err.Error())
	}

	addLogEntry(fmt.Sprintf("ClearBlade Edge version %s downloaded from Github\n", version))
	return nil
}

func installEdge(version string) error {
	var cmdResp interface{}
	var err error
	var msg string

	addLogEntry(fmt.Sprintln("Installing updated Edge..."))

	//Un-tar binary
	log.Printf("[DEBUG] installEdge - Executing command: tar xzvf %s\n", "/tmp/"+edgeDownloadName)
	addLogEntry(fmt.Sprintf("Executing tar command on file %s\n", "/tmp/"+edgeDownloadName))

	if cmdResp, err = executeOSCommand("tar", []string{"xzvf", "/tmp/" + edgeDownloadName}); err != nil {
		msg = "Error encountered executing the tar command"
	} else {
		//Move binary to install location
		log.Printf("[DEBUG] installEdge - Executing command: mv edge-%s %s\n", version, edgeInstallDir+"/edge")
		addLogEntry(fmt.Sprintf("Moving binary to %s\n", edgeInstallDir))

		if cmdResp, err = executeOSCommand("mv", []string{"edge-" + version, edgeInstallDir + "/edge"}); err != nil {
			msg = "Error encountered moving edge binary to " + edgeInstallDir
		} else {
			//chgrp on binary
			log.Printf("[DEBUG] installEdge - Executing command: chgrp root %s\n", edgeInstallDir+"/edge")
			addLogEntry(fmt.Sprintf("Changing the ownership group to root: chgrp root %s\n", edgeInstallDir+"/edge"))

			if cmdResp, err = executeOSCommand("chgrp", []string{"root", edgeInstallDir + "/edge"}); err != nil {
				msg = "Error encountered changing the ownership group to root"
			} else {
				//chown on binary
				log.Printf("[DEBUG] installEdge - Executing command: chown root %s\n", edgeInstallDir+"/edge")
				addLogEntry(fmt.Sprintf("Changing the owner to root: chown root %s\n", edgeInstallDir+"/edge"))

				if cmdResp, err = executeOSCommand("chown", []string{"root", edgeInstallDir + "/edge"}); err != nil {
					msg = "Error encountered changing the owner to root"
				} else {
					//chmod on binary
					log.Printf("[DEBUG] installEdge - Executing command: chmod +x %s\n", edgeInstallDir+"/edge")
					addLogEntry(fmt.Sprintf("Changing permissions: chmod +x %s\n", edgeInstallDir+"/edge"))

					if cmdResp, err = executeOSCommand("chmod", []string{"+x", edgeInstallDir + "/edge"}); err != nil {
						msg = "Error encountered changing permissions"
					} else {
						//Deleting downloaded file
						log.Printf("[DEBUG] installEdge - Executing command: rm %s\n", "/tmp/"+edgeDownloadName)
						addLogEntry(fmt.Sprintf("Deleting downloaded file: rm %s\n", "/tmp/"+edgeDownloadName))

						if cmdResp, err = executeOSCommand("rm", []string{"/tmp/" + edgeDownloadName}); err != nil {
							msg = "Error encountered deleting " + "/tmp/" + edgeDownloadName
						}
					}
				}
			}
		}
	}

	if err != nil {
		errString := msg + ": " + err.Error() + "\n"
		if cmdResp != nil {
			errString = errString + "\nCommand Response: " + cmdResp.(string)
		}
		return errors.New(errString)
	}

	addLogEntry(fmt.Sprintf("Edge version %s installed\n", version))
	return nil
}

func executeOSCommand(osCmd string, args []string) (interface{}, error) {
	cmd := exec.Command(osCmd, args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] executeOSCommand - ERROR executing command: %s\n", err.Error())
		return nil, err
	} else {
		log.Printf("[DEBUG] executeOSCommand - Command response received: %s\n", out.String())

		return out.String(), nil
	}
}

func addErrorToPayload(payload map[string]interface{}, errMsg string) {
	payload["success"] = false
	if payload["error"] == nil || payload["error"] == "" {
		payload["error"] = errMsg
	} else {
		payload["error"] = payload["error"].(string) + "\n" + errMsg
	}
}

// Subscribes to a topic
func subscribe(topic string) (<-chan *mqttTypes.Publish, error) {
	log.Printf("[DEBUG] subscribe - Subscribing to topic %s\n", topic)
	subscription, error := cbBroker.client.Subscribe(topic, cbBroker.qos)
	if error != nil {
		log.Printf("[ERROR] subscribe - Unable to subscribe to topic: %s due to error: %s\n", topic, error.Error())
		return nil, error
	}

	log.Printf("[DEBUG] subscribe - Successfully subscribed to = %s\n", topic)
	return subscription, nil
}

// Publishes data to a topic
func publish(topic string, data string) error {
	log.Printf("[DEBUG] publish - Publishing to topic %s\n", topic)
	error := cbBroker.client.Publish(topic, []byte(data), cbBroker.qos)
	if error != nil {
		log.Printf("[ERROR] publish - Unable to publish to topic: %s due to error: %s\n", topic, error.Error())
		return error
	}

	log.Printf("[DEBUG] publish - Successfully published message to = %s\n", topic)
	return nil
}

func publishResponse(respJson map[string]interface{}) {
	//Create the response topic
	theTopic := topicRoot + "/" + edgeId + "/response"

	respStr, err := json.Marshal(respJson)
	if err != nil {
		log.Printf("[ERROR] publishResponse - ERROR marshalling json response: %s\n", err.Error())
	} else {
		log.Printf("[DEBUG] publishResponse - Publishing response %s to topic %s\n", string(respStr), theTopic)

		//Publish the response
		err = publish(theTopic, string(respStr))
		if err != nil {
			log.Printf("[ERROR] publishResponse - ERROR publishing to topic: %s\n", err.Error())
		}
	}
}

func publishLogs() {
	logsPayload := make(map[string]interface{})
	logsPayload["logs"] = deployLogs

	//Create the response topic
	theTopic := topicRoot + "/" + edgeId + "/logs"

	logsStr, err := json.Marshal(logsPayload)
	if err != nil {
		log.Printf("[ERROR] publishLogs - ERROR marshalling json response: %s\n", err.Error())
	} else {
		log.Printf("[DEBUG] publishLogs - Publishing logs %s to topic %s\n", string(logsStr), theTopic)

		//Publish the logs
		err = publish(theTopic, string(logsStr))
		if err != nil {
			log.Printf("[ERROR] publishLogs - ERROR publishing to topic: %s\n", err.Error())
		}
	}
}

func addLogEntry(log string) {
	deployLogs = append(deployLogs, log)
	publishLogs()
}
