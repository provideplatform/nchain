package main

// func TestRunInstance(t *testing.T) {
// 	bootstrap()

// 	_, err := LaunchAMI("", "", "us-east-1", "ami-f517b88a", "", 1, 1)
// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
// 	}
// }

// func TestRunContainer(t *testing.T) {
// 	bootstrap()

// overrides := map[string]interface{}{
// 	"parity": map[string]interface{}{
// 		"environment": map[string]string{
// 			"CHAIN": "unicorn-v0",
// 		},
// 	},
// }
// 	securityGroupIds := []string{}
// 	subnetIds := make([]string, 0)
// 	_, err := StartContainer("", "", "us-east-1", "providenetwork-node", nil, nil, securityGroupIds, subnetIds, overrides)
// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
// 	}
// }

// func TestGetContainerDetails(t *testing.T) {
// 	bootstrap()

// 	containerDetails, err := GetContainerDetails("", "", "us-east-1", "arn:aws:ecs:us-east-1:085843810865:task/f6988f69-c502-4b67-946c-5ba566053807")
// 	if err == nil {
// 		if len(containerDetails.Tasks) > 0 {
// 			task := containerDetails.Tasks[0]
// 			Log.Debugf("Task: %s", task)

// 		}
// 	}
// }

// func TestGetNetworkInterfaceDetails(t *testing.T) {
// 	bootstrap()

// 	interfaceDetails, err := GetNetworkInterfaceDetails("", "", "us-east-1", "eni-ae4fec96")
// 	if err == nil {
// 		if len(interfaceDetails.NetworkInterfaces) > 0 {
// 			networkInterface := interfaceDetails.NetworkInterfaces[0]
// 			Log.Debugf("Interface: %s", networkInterface)
// 		}
// 	}
// }

// func TestStopContainer(t *testing.T) {
// 	bootstrap()

// 	_, err := StopContainer("", "", "us-east-1", "production", "dfee3958-4768-49ec-b2c3-0d505d0bb557")
// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
// 	}
// }

// func TestAuthorizeSecurityGroupIngress(t *testing.T) {
// 	bootstrap()

// 	group, err := CreateSecurityGroup("", "", "us-east-1", "test group", "security group for testing...", nil)
// 	Log.Debugf("group: %s", group)

// 	tcpPorts := []int64{8050, 8051, 30300}
// 	udpPorts := []int64{30300}
// 	_, err = AuthorizeSecurityGroupIngress("", "", "us-east-1", *group.GroupId, "0.0.0.0/0", tcpPorts, udpPorts)
// 	_, err = AuthorizeSecurityGroupEgressAllPortsAllProtocols("", "", "us-east-1", *group.GroupId)

// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
// 	}
// }

// func TestGetSubnets(t *testing.T) {
// 	bootstrap()

// 	subnets, err := GetSubnets("", "", "us-east-1")
// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
// 	}
// 	Log.Debugf("subnets %s", subnets)
// }

// func TestGetInstanceDetails(t *testing.T) {
// 	bootstrap()

// 	instanceDetails, err := GetInstanceDetails("", "", "us-east-1", "")
// 	if err != nil {
// 		if len(instanceDetails.Reservations) > 0 {
// 			reservation := instanceDetails.Reservations[0]
// 			if len(reservation.Instances) > 0 {
// 				instance := reservation.Instances[0]

// 			}
// 		}
// 	}
// }
