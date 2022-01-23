package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/fortress/csi/driver"
)

func main() {

	var (
		endpoint   = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/"+driver.DefaultDriverName+"/csi.sock", "CSI endpoint")
		token      = flag.String("token", "", "Vultr API Token")
		driverName = flag.String("driver-name", driver.DefaultDriverName, "Name of driver")
		userAgent  = flag.String("user-agent", fmt.Sprintf("csi-fortress/%s", driver.DriverVersion), "Custom user agent")
	)
	flag.Parse()

	d, err := driver.NewDriver(*endpoint, *token, *driverName, driver.DriverVersion, *userAgent)
	if err != nil {
		log.Fatalln(err)
	}

	d.Run()

}
