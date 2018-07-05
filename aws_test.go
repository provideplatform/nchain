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

// 	overrides := map[string]interface{}{
// 		"parity": map[string]interface{}{
// 			"environment": map[string]string{
// 				"CHAIN": "unicorn-v0",
// 			},
// 		},
// 	}
// 	securityGroupIds := []string{}
// 	subnetIds := make([]string, 0)
// 	_, err := StartContainer("", "", "us-east-1", "providenetwork-node", nil, nil, securityGroupIds, subnetIds, overrides)
// 	if err != nil {
// 		t.Fail()
// 		Log.Debugf("%s", err)
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
