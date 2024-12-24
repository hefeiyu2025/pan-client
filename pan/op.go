package pan

import (
	"fmt"
)

var driverConstructorMap = map[DriverType]DriverConstructor{}
var idMap = map[string]Driver{}
var defaultDriverMap = map[DriverType]string{}

func RegisterDriver(driverType DriverType, driver DriverConstructor) {
	driverConstructorMap[driverType] = driver
}
func GetDriver(id string, driverType DriverType, read ConfigRW, write ConfigRW) (Driver, error) {
	if id == "" {
		id = defaultDriverMap[driverType]
	}
	driver, ok := idMap[id]
	if !ok {
		tempDriver := driverConstructorMap[driverType]
		if tempDriver == nil {
			return nil, fmt.Errorf("driver %s not exist", driverType)
		}
		d := tempDriver()
		driverId, err := d.InitByCustom(id, read, write)
		if err != nil {
			return nil, err
		}
		idMap[driverId] = d
		if id == "" {
			defaultDriverMap[driverType] = driverId
		}
		driver = d
	}
	return driver, nil
}

func RemoveDriver(id string) {
	delete(idMap, id)
}
