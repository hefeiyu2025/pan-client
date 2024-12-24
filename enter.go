package pan_client

import (
	"github.com/hefeiyu2025/pan-client/common"
	"github.com/hefeiyu2025/pan-client/pan"
	_ "github.com/hefeiyu2025/pan-client/pan/driver"
)

func GracefulExist() {
	common.Exit()
}
func GetClient(driverType pan.DriverType) (pan.Driver, error) {
	return pan.GetDriver("", driverType, nil, nil)
}

func GetClientById(id string, driverType pan.DriverType) (pan.Driver, error) {
	return pan.GetDriver(id, driverType, nil, nil)
}

func GetClientByRw(id string, driverType pan.DriverType, read pan.ConfigRW, write pan.ConfigRW) (pan.Driver, error) {
	return pan.GetDriver(id, driverType, read, write)
}

func RemoveDriver(id string) {
	pan.RemoveDriver(id)
}
