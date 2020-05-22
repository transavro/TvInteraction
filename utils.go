package main

import (
	"encoding/json"
	"errors"
	"github.com/MarinX/keylogger"
	"github.com/magiconair/properties"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)



// this is the json on which we r gonna decide which process to track.
//[
//{
//"name": "tv.cloudwalker.search",
//"processName": "tv.cloudwalker.alexa",
//"triggerCommands": [
//"am -n tv.cloudwalker.search/.MainActivity"
//],
//"type": "service"
//}
//]


var targetProc Target
var keyProp *properties.Properties

func readFromFile(fileLocation string) (string, error) {
	data, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getEmac() string {
	data, _ := readFromFile("/sys/class/net/eth0/address")
	return strings.TrimSpace(strings.Replace(data, "\n", "", -1))
}

func getTvData() []string {
	var result []string
	//panel
	data, _ := getProp([]string{"ro.cvte.panelname"})
	result = append(result, "Panel="+strings.TrimSpace(strings.Replace(data, "\n", "", -1)))

	//board
	data, _ = getProp([]string{"ro.cvte.boardname", "ro.board.platform"})
	result = append(result, "Board="+strings.TrimSpace(strings.Replace(data, "\n", "", -1)))

	// making model
	data, _ = getProp([]string{"ro.product.model"})
	result = append(result, "Model="+strings.TrimSpace(strings.Replace(data, "\n", "", -1)))

	return result
}

func getProp(keys []string) (string, error) {
	data, err := execute("getprop", keys[0])
	if err != nil {
		return "", err
	}
	if data == "" && len(keys) > 1 {
		data, err = execute("getprop", keys[1])
		if err != nil {
			return "", err
		}
	}
	return data, err
}

func getInstalledAppList() (map[string]string, error) {
	appMap := make(map[string]string)
	output, err := exec.Command("sh", "-c", "pm list packages").Output()
	if err != nil {
		log.Fatal(err)
	}
	appList := strings.TrimSpace(string(output))

	for _, appPackage := range strings.Split(appList, "package:") {
		if len(appPackage) == 0 {
			continue
		}
		output, _ := exec.Command("sh", "-c", "dumpsys package "+strings.TrimSpace(appPackage)+"| grep versionName").Output()
		appMap[strings.TrimSpace(appPackage)] = strings.TrimSpace(strings.Replace(string(output), "versionName=", "", -1))
	}
	return appMap, nil
}

func execute(command string, arg ...string) (string, error) {
	// let's try the pwd command here
	out, err := exec.Command(command, arg...).Output()
	if err != nil {
		return "", err
	}
	return string(out[:]), nil
}

func getTvSources() []string {
	result, err := execute("sh", "-c", "cat /system/cvte.prop | grep ro.CVT_EN_SOURCE")
	if err != nil {
		log.Println(err.Error())
	}
	var tvSources []string
	for _, s := range strings.Split(result, "\n") {
		if strings.Contains(s, "=1") {
			if strings.Contains(s, "SUPPORT") {
				tvSources = append(tvSources, strings.TrimSuffix(strings.Trim(s, "ro.CVT_EN_SOURCE_SUPPORT_"), "=1"))
			} else {
				tvSources = append(tvSources, strings.TrimSuffix(strings.Trim(s, "ro.CVT_EN_SOURCE_"), "=1"))
			}
		}
	}
	return tvSources
}

func readKeyCodeFile(){
	logrus.Println("Reading Key code file")
	if path, err := getProp([]string{"cw.keycode.path"}); err != nil {
		logrus.Error( "Error while getting KeyCode Path file.", err)
	} else if len(path) > 2 {
		logrus.Println("PATH MILA "+path)
		keyProp = properties.MustLoadFile(strings.TrimSuffix(path, "\n"), properties.UTF8)
	}else {
		logrus.Error("Keycode file not found.")
	}
}

func readMonitorFile(){
	if path, err := getProp([]string{"cw.monitor.path"}); err != nil {
		logrus.Error( "error while reading monitor file path", err)
	} else if len(path) > 2 {
		if jsonFile, err := os.Open(strings.TrimSuffix(path, "\n")); err != nil {
			logrus.Error("Error while opening the Monitor file.", err)
		}else {

			defer jsonFile.Close()
			bytevalue, _ := ioutil.ReadAll(jsonFile)
			if err = json.Unmarshal(bytevalue, &targetProc); err != nil {
				logrus.Error("Error while decoding data to json.", err)
			}
		}
	}
}


func getKeyEvent() {

	if keyProp == nil {
		logrus.Info("Keycode Properties not found so will not get the Keyevents.")
		return
	}else if keyProp.Len() == 0 {
		logrus.Info("Keycode Properties doesnt contain any property. so will not get the Keyevents.")
		return
	}

	var found bool
	var cmd string
	keys := keyProp.Keys()

	logrus.Info("In Key Event")
	k, err := keylogger.New("/dev/input/event1")
	if err != nil {
		logrus.Error( err)
		return
	}
	defer k.Close()

	events := k.Read()
	logrus.Info("reading file")
	// range of events
	for e := range events {
		if e.Type == keylogger.EvKey && e.KeyPress() {
			logrus.Info(e.Code)
			found = false
			for _, key := range keys {
				if key == strconv.Itoa(int(e.Code)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			cmd =  keyProp.MustGetString(strconv.Itoa(int(e.Code)))
			if len(cmd) > 0 {
				cmd = strings.TrimSpace(cmd)
				temp := strings.Fields(cmd)
				if err = exec.Command(temp[0], temp[1:]...).Run(); err != nil {
					logrus.Infof("Error while triggering keycode %d = %s", e.Code, err.Error())
				}
			}
		}
	}
}




func preCheck(appName, procType string) (bool, error) {
	var (
		err error
		ans []byte
	)

	if strings.ToLower(procType) == "app"{
		if ans, err = exec.Command("sh", "-c", "pm list packages |  grep " + appName).Output(); err != nil {
			return false, err
		} else if len(ans) > 0 {
			return true, nil
		}else {
			log.Println(len(ans))
			return false, nil
		}
	}else if strings.ToLower(procType) == "bin" {
		if ans, err = exec.Command("which",  appName).Output(); err != nil {
			return false, err
		}else if len(ans) > 0 {
			return true, nil
		}else {
			return false, nil
		}
	}else {
		return false, errors.New("invalid Type")
	}
}


func serviceMonitor(proc struct {
	Name            string   `json:"name"`
	ProcessName     string   `json:"processName"`
	TriggerCommands []string `json:"triggerCommands"`
	Type            string   `json:"type"`
}) {

	// first check if the app or binary is there or not in the system then only  monitor
	if isInstalled, err := preCheck(proc.Name, proc.Type); err != nil {
		log.Println("Error while prechecking ", err.Error())
		return
	}else if isInstalled == false {
		log.Printf("%s is not there in the system so not monitoring.\n", proc.Name)
		return
	}

	for {
		log.Printf("###%s Monitor###\n", proc.Name)
		if op, err := exec.Command("pidof", proc.ProcessName).Output(); err != nil || len(op) == 0{

			//Service is not on
			log.Printf("###%s Restarting..###\n", proc.Name)

			//SUDO
			err := exec.Command("xu" ,"7411").Run()
			if err != nil {
				log.Println("Error while xu ==> "+err.Error())
			}
			for _, command := range proc.TriggerCommands {
				if err = exec.Command("sh", "-c", command).Run(); err != nil {
					log.Println("Error while registering ==> "+err.Error())
				}
			}
		} else {
			log.Printf("###%s is ON###", proc.Name)
		}
		time.Sleep(50 * time.Second)
	}
}