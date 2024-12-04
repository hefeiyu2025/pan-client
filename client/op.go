package client

import (
	"fmt"
)

var driverConstructorMap = map[DriverType]DriverConstructor{}
var driverMap = map[DriverType]Driver{}

func RegisterDriver(driverType DriverType, driver DriverConstructor) {
	driverConstructorMap[driverType] = driver
}
func GetDriver(driverType DriverType) (Driver, error) {
	driver, ok := driverMap[driverType]
	if !ok {
		tempDriver := driverConstructorMap[driverType]
		if tempDriver == nil {
			return nil, fmt.Errorf("driver %s not exist", driverType)
		}
		d := tempDriver()
		err := d.Init()
		if err != nil {
			return nil, err
		}
		driverMap[driverType] = d
		driver = d
	}
	return driver, nil
}
