package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

// Target Timezone
//var TargetLocation = "Europe/Bucharest"

// CameraYaml holds the configuration for a single camera
type CameraYaml struct {
	ID           string  `yaml:"id"`
	Direction    string  `yaml:"direction"`
	AuthKey      string  `yaml:"authkey"`
	RateVariance float64 `yaml:"rate-variance"`
	Rate         float64 `yaml:"rate"`
}

// Config holds the configuration for the application including an array of cameras
type Config struct {
	TargetLocation string       `yaml:"target-location"`
	ConnectorHost  string       `yaml:"connector-host"`
	CarImagePath   string       `yaml:"car-image-path"`
	PlateImagePath string       `yaml:"plate-image-path"`
	Cameras        []CameraYaml `yaml:"cameras"`
	Debug          bool         `yaml:"debug"`
}

type camera struct {
	// Configured
	connHost       string
	id             string
	direction      string //
	rate           float64
	rateVariance   float64
	authKey        string //
	targetLocation string
	plateImagePath string
	carImagePath   string
	debug          bool
	// Derived
	idNum           int
	numberCounter   uint32
	intervalAverage float64
	intervalRange   float64
	minimumInterval float64
}

func (c *camera) init() {
	c.idNum, _ = strconv.Atoi(c.id)
	c.intervalAverage = 3600.0 / (c.rate) // average interval between vehicles
	c.intervalRange = c.intervalAverage * (c.rateVariance / 100.0)
	c.minimumInterval = c.intervalAverage - c.intervalRange
	c.intervalRange *= 2
}

func (c *camera) getNumber(vehicleClass string) string {
	id := fmt.Sprintf("%2.2d", c.idNum)
	count := fmt.Sprintf("%9.9d", c.numberCounter)
	var number = ""
	if vehicleClass == "Car" {
		number = "C"
	} else if vehicleClass == "Truck" {
		number = "T"
	} else if vehicleClass == "Bus" {
		number = "B"
	} else if vehicleClass == "Motorbike" {
		number = "M"
	} else if vehicleClass == "Van" {
		number = "V"
	} else if vehicleClass == "SUV/Pickup" {
		number = "S"
	}
	if number != "" {
		number += id + count
		c.numberCounter++
	} else {
		number = "Unknown"
	}
	return number
}

func getProbability() string {
	prob := rand.Float64()
	probStr := fmt.Sprintf("%1.1f", prob)
	return probStr
}

func getCountryCode() string {
	prob := rand.Float64()
	cc := ""
	if prob < 0.98 {
		cc = "RO"
	} else if prob < 0.995 {
		cc = "D"
	} else {
		cc = "H"
	}
	return cc
}

// Get random vehicle class based upon the following
// weighted distribution. The string values are what
// are sent in the message to the client.
func getVehicleClass() string {
	prob := rand.Float64()
	vc := ""
	if prob < 0.6 {
		vc = "Car" // 60%
	} else if prob < 0.7 {
		vc = "Truck" // 10%
	} else if prob < 0.8 {
		vc = "Bus" // 10%
	} else if prob < 0.9 {
		vc = "Motorbike" // 10%
	} else if prob < 0.95 {
		vc = "Van" // 5%
	} else if prob < 0.99 {
		vc = "SUV/Pickup" // 4%
	} else {
		vc = "Unknown" // 1%
	}
	return vc
}

func getVehicleColor(vehicle string) string {
	prob := rand.Float64()
	vc := ""
	if vehicle == "Car" {
		// black 23 white 19%,  grey 18% silver 15%, blue 10, red - 10 brown-2, gold - 1% green 1
		if prob < 0.23 {
			vc = "Black"
		} else if prob < 0.42 {
			vc = "White"
		} else if prob < 0.60 {
			vc = "Grey"
		} else if prob < 0.75 {
			vc = "Silver"
		} else if prob < 0.85 {
			vc = "Blue"
		} else if prob < 0.95 {
			vc = "Red"
		} else if prob < 0.97 {
			vc = "Brown"
		} else if prob < 0.98 {
			vc = "Gold"
		} else if prob < 0.99 {
			vc = "Green"
		} else {
			vc = "Unknown"
		}
	} else if vehicle == "Truck" {
		vc = "White"
	} else if vehicle == "Bus" {
		vc = "White"
	} else if vehicle == "Motorbike" {
		if prob < 0.40 {
			vc = "Black"
		} else if prob < 0.80 {
			vc = "White"
		} else {
			vc = "Red"
		}
	} else if vehicle == "Van" {
		if prob < 0.40 {
			vc = "White"
		} else if prob < 0.80 {
			vc = "Silver"
		} else {
			vc = "Grey"
		}
	} else if vehicle == "SUV/Pickup" {
		if prob < 0.40 {
			vc = "White"
		} else if prob < 0.80 {
			vc = "Silver"
		} else {
			vc = "Grey"
		}
	} else {
		vc = "Unknown"
	}
	return vc
}

func getTargetTimestamp(targetLocation string) string {
	loc, err := time.LoadLocation(targetLocation)
	if err != nil {
		fmt.Println("Error LoadLocation failed: Cause: ", err.Error())
	}
	now := time.Now().In(loc)
	timeStr := now.Format("2006-01-02_15:04:05")
	return timeStr
}

func getTransactionKey() string {
	uuid := strings.Replace(uuid.New().String(), "-", "", -1)
	return uuid
}

func (cam *camera) send() {

	vehicleClass := getVehicleClass()
	vehicleColor := getVehicleColor(vehicleClass)
	number := cam.getNumber(vehicleClass)
	countryCode := getCountryCode()
	firstSeen := getTargetTimestamp(cam.targetLocation)
	lastSeen := firstSeen
	probability := getProbability()
	transactionKey := getTransactionKey()

	// Default values
	gpsLatitude := "0"
	gpsLongitude := "0"
	haveCompanion := "0"
	weight := "0"
	speed := "0"
	triggerKey := "none"

	md5Sum := md5.New()
	hashString := cam.id + number + countryCode + firstSeen
	hashString += lastSeen + probability + transactionKey + cam.direction
	hashString += gpsLatitude + gpsLongitude + haveCompanion + cam.authKey
	//With addition of Vehicle Class support weight and speed were removed from the
	// authorization check.

	io.WriteString(md5Sum, hashString)
	sum := md5Sum.Sum(nil)
	hexString := fmt.Sprintf("%x", sum)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Authorization
	writer.WriteField("id", cam.id)
	writer.WriteField("number", number)
	writer.WriteField("country_code", countryCode)
	writer.WriteField("first_seen", firstSeen)
	writer.WriteField("last_seen", lastSeen)
	writer.WriteField("probability", probability)
	writer.WriteField("transactionkey", transactionKey)
	writer.WriteField("direction", cam.direction)
	writer.WriteField("gps_latitude", gpsLatitude)
	writer.WriteField("gps_longitude", gpsLongitude)
	writer.WriteField("have_companion", haveCompanion)
	writer.WriteField("auth", hexString)

	writer.WriteField("vehicle_class", vehicleClass)
	writer.WriteField("vehicle_color", vehicleColor)

	// Other default values
	writer.WriteField("weight", weight)
	writer.WriteField("speed", speed)
	writer.WriteField("triggerKey", triggerKey)

	// Images now.
	// car_image
	var carImageFileName string
	var plateImageFileName string
	if isRunningInDockerContainer() {
		// Container path
		carImageFileName = "/data/car_image"
		plateImageFileName = "/data/plate_image"
	} else {
		// Absolute path
		carImageFileName = cam.carImagePath
		plateImageFileName = cam.plateImagePath
	}
	carImageFile, errCarOpen := os.Open(carImageFileName)
	if errCarOpen != nil {
		fmt.Println("Error opening car image file. Cause: ", errCarOpen.Error())
		return
	}
	defer carImageFile.Close()
	carImagePart, errCarForm := writer.CreateFormFile("car_image", filepath.Base(carImageFileName))
	if errCarForm != nil {
		fmt.Println("Error creating car image form file. Cause: ", errCarForm.Error())
		return
	}
	_, errCarCopy := io.Copy(carImagePart, carImageFile)
	if errCarCopy != nil {
		fmt.Println("Error copying car file contents to image part. Cause: ", errCarCopy.Error())
		return
	}

	// plate_image
	plateImageFile, errPlateOpen := os.Open(plateImageFileName)
	if errPlateOpen != nil {
		fmt.Println("Error opening plate image file. Cause: ", errPlateOpen.Error())
		return
	}
	defer plateImageFile.Close()
	plateImagePart, errPlateForm := writer.CreateFormFile("plate_image", filepath.Base(plateImageFileName))
	if errPlateForm != nil {
		fmt.Println("Error creating plate image form file. Cause: ", errPlateForm.Error())
		return
	}
	_, errPlateCopy := io.Copy(plateImagePart, plateImageFile)
	if errPlateCopy != nil {
		fmt.Println("Error copying plate file contents to image part. Cause: ", errPlateCopy.Error())
		return
	}

	errWriterClose := writer.Close()
	if errWriterClose != nil {
		fmt.Println("Error closing writer. Cause: ", errWriterClose.Error())
		return
	}

	if cam.debug {
		fmt.Println("body:\n", body)
	}
	uri := "http://" + cam.connHost + ":8879/lpr"
	req, newReqErr := http.NewRequest("POST", uri, body)
	if newReqErr != nil {
		fmt.Println("Error creating new request. Cause: ", newReqErr.Error())
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Close = true // Fixes sporatic EOF errors?

	client := &http.Client{}

	resp, errDo := client.Do(req)

	if errDo != nil {
		fmt.Println("Error in sending request. Cause: ", errDo.Error())
	} else {
		defer resp.Body.Close()
		body := &bytes.Buffer{}
		_, errBodyRead := body.ReadFrom(resp.Body)
		if errBodyRead != nil {
			fmt.Println("Error reading from response. Cause: ", errBodyRead)
		}

		if resp.StatusCode == 200 {
			if body.String() != "bb1e8f805814a0b8e465601346872377" {
				fmt.Println("Status code is ok. Response body is ", body)
			}
		} else {
			fmt.Printf("Status code is %d. Response body is %s\n", resp.StatusCode, body.String())
		}
	}
}

func (c *camera) Run() {
	for {
		pre := time.Now().UnixNano()
		c.send()
		post := time.Now().UnixNano()
		tdiff := post - pre
		fmt.Printf("%d.%9.9d\n", tdiff/1000000000, tdiff%1000000000)
		value := c.minimumInterval + c.intervalRange*rand.Float64()
		sleepDuration := time.Duration(value * 1000000000)
		time.Sleep(sleepDuration)
	}
}

func isRunningInDockerContainer() bool {
	// docker creates a .dockerenv file at the root
	// of the directory tree inside the container.
	// if this file exists then the viewer is running
	// from inside a container so return true

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

func main() {

	var configFile string
	if isRunningInDockerContainer() {
		// Container path
		configFile = "/data/" + os.Getenv("CONFIG")
	} else {
		// Absolute path
		configFile = os.Getenv("CONFIG")
	}

	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configFile)
	if err != nil {
		panicMsg := fmt.Sprintf("Error in opening configuration file, %s. Cause: %s\n", configFile, err.Error())
		panic(panicMsg)
	}
	defer file.Close()
	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		fmt.Println("Error in decode YAML from file. Cause: ", err.Error())
	}

	if !isRunningInDockerContainer() {
		if config.CarImagePath == "" {
			fmt.Println("Car Image is not defined in configuration file")
			return
		}
		if config.PlateImagePath == "" {
			fmt.Println("Plate Image is not defined in configuration file")
			return
		}
	}
	for _, c := range config.Cameras {
		var cam = camera{id: c.ID, connHost: config.ConnectorHost, authKey: c.AuthKey,
			direction: c.Direction, rate: c.Rate, rateVariance: c.RateVariance,
			targetLocation: config.TargetLocation, carImagePath: config.CarImagePath,
			plateImagePath: config.PlateImagePath, debug: config.Debug}
		cam.init()
		go cam.Run()
	}

	for {
		time.Sleep(1 * time.Second)
	}

}
