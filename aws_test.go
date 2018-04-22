package main

import "testing"

func TestRunInstance(t *testing.T) {
	bootstrap()

	_, err := LaunchAMI("", "", "us-east-1", "ami-f517b88a", "", 1, 1)
	if err != nil {
		t.Fail()
		Log.Debugf("%s", err)
	}
}

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
