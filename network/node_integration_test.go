// +build integration

package network_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	awswrapper "github.com/kthomas/go-aws-wrapper"
	uuid "github.com/kthomas/go.uuid"
	"github.com/onsi/gomega/gstruct"
	provide "github.com/provideservices/provide-go"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var steLock = &sync.Mutex{}
var ctxLocks = map[string]*network.Node{}

func TestNodesIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Suite")
}

func ptrTo(s string) *string {
	return &s
}

func dbconn() *gorm.DB {
	return dbconf.DatabaseConnection()
}

func nodeExists(role, status string) (*network.Node, bool) {
	node := &network.Node{}
	dbconn().Where("role = ? AND status = ?", role, status).Last(&node)
	return node, node.ID != uuid.Nil
}

func createValidatorNode(ctx, awsKey, awsSecret, awsRegion string, searchDB bool, awsTasksStartedIds *[2]*string) (node *network.Node, err error) {
	steLock.Lock()
	defer steLock.Unlock()

	ctxLockKey := fmt.Sprintf("%s-createValidatorNode", ctx)
	if n, lockOk := ctxLocks[ctxLockKey]; lockOk {
		err = fmt.Errorf("Already have established a context lock for '%s'; bailing out without deploying subsequent validator node", ctx) // we may find it useful to have a lock-per-context so instead of making it bool I just made another lock which doubles as a lock and a semaphore :)

		return n, err
	}

	node = &network.Node{}

	if searchDB {
		// db.Where("role = ? AND status = ?", ).Last(&node)
		if _node, nodeOk := nodeExists("validator", "running"); nodeOk {
			ctxLocks[ctxLockKey] = _node
			return _node, nil
		}
	}

	// node = &network.Node{}
	ctxLocks[ctxLockKey] = node

	fmt.Printf("locks: %v\n", ctxLocks)
	// return

	nodeConfig := fmt.Sprintf(`{"config":{"credentials":{"aws_access_key_id":"%s","aws_secret_access_key":"%s"},"engine_id":"aura","engines":"[{\"id\":\"aura\",\"name\":\"Authority Round\",\"enabled\":true},{\"id\":\"clique\",\"name\":\"Clique\",\"enabled\":true}]","env":{"CHAIN_SPEC_URL":"https://www.dropbox.com/s/515w1hxayztx80e/spec.json?dl=1","ENGINE_SIGNER":"0x1e23ce07ebC8d1f515C5f04101018e8A9Ab9353f","NETWORK_ID":"1537367446","TRACING":"on","PRUNING":"archive","FAT_DB":"on","LOGGING":"verbose","CHAIN":"mk"},"protocol_id":"poa","provider_id":"docker","providers":"[{\"id\":\"docker\",\"name\":\"Docker\",\"img_src_dark\":\"https://s3.amazonaws.com/provide.services/img/docker-dark.png\",\"img_src_light\":\"https://s3.amazonaws.com/provide.services/img/docker-light.png\",\"enabled\":true,\"img_src\":\"/assets/docker-c9b76b9cdf162dd88c2c685c5f2fa45deea5209de420b220f2d35273f33a397a.png\"}]","rc.d":"{\n  \"CHAIN_SPEC_URL\": \"https://www.dropbox.com/s/xuihpmm72o5m9ir/spec.json?dl=1\",\n  \"ENGINE_SIGNER\": \"0x41f4b57D711c93a22F0Ac580a842eac52E4bC505\",\n  \"NETWORK_ID\": \"1537367446\",\n  \"TRACING\": \"on\",\n  \"PRUNING\": \"archive\",\n  \"FAT_DB\": \"on\",\n  \"LOGGING\": \"verbose\",\n  \"CHAIN\": \"kt\"\n}","region":"%s","role":"validator","roles":"[{\"id\":\"peer\",\"name\":\"Peer\",\"config\":{\"allows_multiple_deployment\":true,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"JSON_RPC_URL\\\": null, \\\"NETWORK_ID\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":2},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"validator\",\"name\":\"Validator\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"ENGINE_SIGNER\\\": null, \\\"NETWORK_ID\\\": null, \\\"ENGINE_SIGNER_PRIVATE_KEY\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"explorer\",\"name\":\"Block Explorer\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"JSON_RPC_URL\\\": null}\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"docker\"]}]","target_id":"aws"},"network_id":"36a5f8e0-bfc1-49f8-a7ba-86457bb52912"}`, awsKey, awsSecret, awsRegion)

	id, err := uuid.FromString("bd7f1b2d-45f6-4399-afcf-7836986d3dc9") // same uuid as in networks_test.sql
	// Expect(err).NotTo(HaveOccurred())

	node.UserID = &id
	err = json.Unmarshal([]byte(nodeConfig), node)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	Expect(err).NotTo(HaveOccurred())

	fmt.Printf("=== CREATE node: %v", node)
	var cfg map[string]interface{}
	json.Unmarshal(*node.Config, &cfg)
	fmt.Printf("   with config: %#v\n", cfg)

	res := node.Create()
	fmt.Printf("%t\n", res)

	if res {
		wait(time.Duration(10)*time.Second, ptrTo("to get validator node config..."))

		node.Reload()
		taskIds := keyFromConfig(node.Config, "target_task_ids").([]interface{})
		fmt.Printf("task ids: %v\n", taskIds)
		taskId := taskIds[0].(string)
		awsTasksStartedIds[0] = &taskId

		wait(time.Duration(6)*time.Minute, ptrTo("to deploy validator node..."))
	}

	return node, nil
}

func createPeerNode(ctx, awsKey, awsSecret string) (node *network.Node, err error) {
	steLock.Lock()
	defer steLock.Unlock()

	ctxLockKey := fmt.Sprintf("%s-createPeerNode", ctx)
	if n, lockOk := ctxLocks[ctxLockKey]; lockOk {
		err = fmt.Errorf("Already have established a context lock for '%s'; bailing out without deploying subsequent peer node", ctx) // we may find it useful to have a lock-per-context so instead of making it bool I just made another lock which doubles as a lock and a semaphore :)
		return n, err
	}
	node = &network.Node{Role: common.StringOrNil("peer")}

	// db := dbconf.DatabaseConnection()
	// db.Where("role = ? AND status = ?", "peer", "running").Last(&node)

	if _node, nodeOk := nodeExists("peer", "running"); nodeOk {
		fmt.Printf("=== NODE found ")
		return _node, nil
	}

	fmt.Printf("node: %v\n", node)
	ctxLocks[ctxLockKey] = node

	fmt.Printf("locks: %v\n", ctxLocks)

	// return

	// nodeConfig := fmt.Sprintf(`{"config":{"credentials":{"aws_access_key_id":"%s","aws_secret_access_key":"%s"},"engine_id":"aura","engines":"[{\"id\":\"aura\",\"name\":\"Authority Round\",\"enabled\":true},{\"id\":\"clique\",\"name\":\"Clique\",\"enabled\":true}]","env":{"CHAIN_SPEC_URL":"https://www.dropbox.com/s/515w1hxayztx80e/spec.json?dl=1","NETWORK_ID":"1537367446","TRACING":"on","PRUNING":"archive","FAT_DB":"on","LOGGING":"verbose","CHAIN":"mk"},"protocol_id":"poa","provider_id":"docker","providers":"[{\"id\":\"docker\",\"name\":\"Docker\",\"img_src_dark\":\"https://s3.amazonaws.com/provide.services/img/docker-dark.png\",\"img_src_light\":\"https://s3.amazonaws.com/provide.services/img/docker-light.png\",\"enabled\":true,\"img_src\":\"/assets/docker-c9b76b9cdf162dd88c2c685c5f2fa45deea5209de420b220f2d35273f33a397a.png\"}]","rc.d":"{\n  \"CHAIN_SPEC_URL\": \"https://www.dropbox.com/s/xuihpmm72o5m9ir/spec.json?dl=1\",\n  \"NETWORK_ID\": \"1537367446\",\n  \"TRACING\": \"on\",\n  \"PRUNING\": \"archive\",\n  \"FAT_DB\": \"on\",\n  \"LOGGING\": \"verbose\",\n  \"CHAIN\": \"kt\"\n}","region":"us-west-2","role":"peer","roles":"[{\"id\":\"peer\",\"name\":\"Peer\",\"config\":{\"allows_multiple_deployment\":true,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"JSON_RPC_URL\\\": null, \\\"NETWORK_ID\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":2},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"validator\",\"name\":\"Validator\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"NETWORK_ID\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"explorer\",\"name\":\"Block Explorer\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"JSON_RPC_URL\\\": null}\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"docker\"]}]","target_id":"aws"},"network_id":"36a5f8e0-bfc1-49f8-a7ba-86457bb52912"}`, awsKey, awsSecret)
	nodeConfig := fmt.Sprintf(`{"config":{"credentials":{"aws_access_key_id":"%s","aws_secret_access_key":"%s"},"engine_id":"aura","env":{"CHAIN_SPEC_URL":"https://www.dropbox.com/s/515w1hxayztx80e/spec.json?dl=1","NETWORK_ID":"1537367446"},"protocol_id":"poa","provider_id":"docker","region":"us-west-2","role":"peer","target_id":"aws"},"network_id":"36a5f8e0-bfc1-49f8-a7ba-86457bb52912"}`, awsKey, awsSecret)

	err = json.Unmarshal([]byte(nodeConfig), node)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	Expect(err).NotTo(HaveOccurred())

	id, err := uuid.FromString("bd7f1b2d-45f6-4399-afcf-7836986d3dc9") // same uuid as in networks_test.sql
	// Expect(err).NotTo(HaveOccurred())

	node.UserID = &id

	err = json.Unmarshal([]byte(nodeConfig), node)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	// Expect(err).NotTo(HaveOccurred())

	fmt.Println("=============================")
	fmt.Printf("second node: %#v\n", node)
	var cfg map[string]interface{}
	json.Unmarshal(*node.Config, &cfg)
	fmt.Printf("node config: %#v\n", cfg)

	fmt.Printf("creating node: ")

	res := node.Create()
	fmt.Printf("%t\n", res)
	if !res {
		fmt.Printf("errors: \n")
		for i, e := range node.Errors {
			fmt.Printf("  %v. %v", i, *e.Message)
		}
	}

	if res {
		wait(time.Duration(8)*time.Minute, ptrTo("to deploy second node..."))
		node.Reload()
	}

	return node, nil
}

func releaseContextMutex(ctx string, t string) (err error) {
	var ctxLockKey string
	if t == "peer" {
		ctxLockKey = fmt.Sprintf("%s-createPeerNode", ctx)
	} else if t == "validator" {
		ctxLockKey = fmt.Sprintf("%s-createValidatorNode", ctx)
	}

	delete(ctxLocks, ctxLockKey)
	return err
}

func contextMutexReleased(ctx string, t string) bool {
	var ctxLockKey string
	if t == "peer" {
		ctxLockKey = fmt.Sprintf("%s-createPeerNode", ctx)
	} else if t == "validator" {
		ctxLockKey = fmt.Sprintf("%s-createValidatorNode", ctx)
	}
	_, lockOk := ctxLocks[ctxLockKey]
	return !lockOk
}

func removeNode(ctx string, node *network.Node) (err error) {
	node.Reload()
	node.Delete()
	// Expect(node.Status).To(gstruct.PointTo(Equal("deprovisioning")))

	wait(time.Duration(15)*time.Second, nil)
	node.Reload()
	Expect(node.Status).To(gstruct.PointTo(Equal("terminated")))

	return nil
}

func startStatsDaemon(n *network.Network) {
	n.Status(false)
}

func keyFromConfig(c *json.RawMessage, key string) interface{} {
	config := map[string]interface{}{}
	json.Unmarshal(*c, &config)
	v, _ := config[key]
	return v
}

func wait(d time.Duration, msg *string) {
	if msg == nil {
		msg = ptrTo("")
	}
	time.Sleep(d)
	fmt.Printf("=== WAIT %v %v\n", d, *msg)
}

// func ecsClient(awsKey, awsSecret, awsRegion string) *ecs.ECS, error {
// 	ecsClient, err := awswrapper.NewECS(awsKey, awsSecret, awsRegion)
// 	return ecsClient, err
// }

// print out tasks, loadbalancers, security groups
func printAWSresources(awsKey, awsSecret, awsRegion, title string) (lbs, sgs, tasks []*string) {
	fmt.Printf("\n\n=== AWS %v: \n", title)

	ecsClient, _ := awswrapper.NewECS(awsKey, awsSecret, awsRegion)

	fmt.Printf("===    tasks on aws:\n")
	td, _ := ecsClient.ListTasks(&ecs.ListTasksInput{
		Cluster: ptrTo("production"),
	})
	tasks = make([]*string, len(td.TaskArns))
	for i, ts := range td.TaskArns {
		tasks[i] = ts
		fmt.Printf("     %v\n", *ts)
	}

	elbClient, _ := awswrapper.NewELBv2(awsKey, awsSecret, awsRegion)

	lbResponse, err := elbClient.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		// LoadBalancerArns: []*string{},
		// LoadBalancerNames: []*string{},
	})
	// lbs, err := awswrapper.GetLoadBalancers(awsKey, awsSecret, awsRegion, nil)
	if err != nil {
		fmt.Printf("==! LB error: %v\n", err)
	}
	fmt.Printf("=== load balancers: \n")
	lbs = make([]*string, len(lbResponse.LoadBalancers))
	for i, lb := range lbResponse.LoadBalancers {
		lbs[i] = lb.LoadBalancerArn
		fmt.Printf("    %v %v\n", *lb.LoadBalancerName, lb.SecurityGroups)
	}

	sGroups, err := awswrapper.GetSecurityGroups(awsKey, awsSecret, awsRegion)
	if err != nil {
		fmt.Printf("==! SG error: %v\n", err)
	}
	fmt.Printf("=== security groups: \n")
	sgs = make([]*string, len(sGroups.SecurityGroups))
	for i, sg := range sGroups.SecurityGroups {
		sgs[i] = sg.GroupId
		fmt.Printf("    %v %v\n", *sg.GroupId, *sg.GroupName)
	}

	return
}

func printDBResources(db *gorm.DB, title string) (nodeIds, lbIds []*uuid.UUID) {
	fmt.Printf("\n\n=== DB %v: \n", title)

	nodes := []*network.Node{}
	db.Find(&nodes)

	loadBalancers := []*network.LoadBalancer{}
	db.Find(&loadBalancers)
	// fmt.Printf("===    load balancers: %v\n", loadBalancers)

	nodeIds = make([]*uuid.UUID, len(nodes))
	lbIds = make([]*uuid.UUID, len(loadBalancers))
	for i, node := range nodes {
		nodeIds[i] = &node.ID
	}
	for i, lb := range loadBalancers {
		lbIds[i] = &lb.ID
	}

	return
}

func arrContainsStr(arr []*string, str *string) bool {
	for _, s := range arr {
		if *s == *str {
			return true
		}
	}
	return false
}

func arrContainsUUID(arr []*uuid.UUID, uid *uuid.UUID) bool {
	for _, u := range arr {
		if *u == *uid {
			return true
		}
	}
	return false
}

func extraStrings(a1, a2 []*string) []*string {
	a := []*string{}
	for _, s := range a2 {
		if !arrContainsStr(a1, s) {
			a = append(a, s)
		}
	}
	return a
}

func extraUUIDs(a1, a2 []*uuid.UUID) []*uuid.UUID {
	a := []*uuid.UUID{}
	for _, s := range a2 {
		if !arrContainsUUID(a1, s) {
			a = append(a, s)
		}
	}
	return a
}

func taskStop(ecsClient *ecs.ECS, taskId *string) {
	fmt.Printf("=== NEED TO STOP TASK: %v\n", *taskId)

	_, err := ecsClient.StopTask(&ecs.StopTaskInput{
		Cluster: ptrTo("production"),
		Reason:  ptrTo("test cleanup"),
		Task:    taskId,
	})
	if err == nil {
		fmt.Printf("    TASK STOPPED: %v\n", *taskId)
	} else {
		fmt.Printf("error: %v\n", err)
	}
}

var _ = Describe("Node", func() {
	var n = &network.Network{}
	var networkId uuid.UUID
	db := dbconf.DatabaseConnection()

	awsKey := os.Getenv("TEST_AWS_ACCESS_KEY_ID")
	awsSecret := os.Getenv("TEST_AWS_SECRET_ACCESS_KEY")
	awsRegion := "us-west-2"

	// var nodesGenerated = []*network.Node{}
	var awsTasksStartedIds = [2]*string{}
	var lbStartedIds = []*string{}
	// var sgStartedIds = []*string{}

	var preTestAWSTasks []*string
	var preTestAWSLBs []*string
	var preTestAWSSGs []*string
	var preTestNodeIds []*uuid.UUID
	var preTestLBIds []*uuid.UUID

	BeforeSuite(func() {
		// lbs, _ := n.LoadBalancers(dbconf.DatabaseConnection(), nil, nil)
		// Expect(len(lbs)).To(Equal(0)) // assert zero-state...

		// awsRun := os.Getenv("AWS_RUN")

		// if awsRun == "1" {
		networkId, _ = uuid.FromString("36a5f8e0-bfc1-49f8-a7ba-86457bb52912")
		db.First(&n, "id = ?", networkId)

		if awsKey != "" && awsSecret != "" {
			preTestAWSLBs, preTestAWSSGs, preTestAWSTasks = printAWSresources(awsKey, awsSecret, awsRegion, "START")
			preTestNodeIds, preTestLBIds = printDBResources(db, "START")
		}
		// }
	})

	AfterSuite(func() {
		// get amazon nodes
		// get amazon lbs, sgs, tgs
		// remove test instances and settings

		if awsKey != "" && awsSecret != "" {
			network.EvictNetworkStatsDaemon(n)
			wait(time.Duration(5)*time.Second, nil)

			fmt.Printf("===    tasks started during test: %v\n", awsTasksStartedIds)
			ecsClient, _ := awswrapper.NewECS(awsKey, awsSecret, awsRegion)
			for _, taskId := range awsTasksStartedIds {

				if taskId != nil {

					// taskIdParts := strings.Split(*taskId, "/")
					// td, _ := awswrapper.GetTaskDefinition(awsKey, awsSecret, awsRegion, taskIdParts[1])

					td, _ := ecsClient.DescribeTasks(&ecs.DescribeTasksInput{
						Cluster: ptrTo("production"),
						Tasks:   []*string{taskId},
					})
					// fmt.Printf("=== task %v: %v\n", *taskId, td)

					if len(td.Tasks) > 0 {
						fmt.Printf("=== task %v status: %v\n", *taskId, *(*td.Tasks[0]).LastStatus)

						//if *(*td.Tasks[0]).LastStatus == "RUNNING" {
						taskStop(ecsClient, taskId)
						//}
					}
				} else {
					fmt.Printf("task: %v\n", taskId)
				}
			}

			postTestAWSLBs, postTestAWSSGs, postTestAWSTasks := printAWSresources(awsKey, awsSecret, awsRegion, "LEFTOVERS")
			extraLBs := extraStrings(preTestAWSLBs, postTestAWSLBs)
			extraSGs := extraStrings(preTestAWSSGs, postTestAWSSGs)
			extraTasks := extraStrings(preTestAWSTasks, postTestAWSTasks)

			fmt.Printf("===   lbs started during test: %v\n", len(lbStartedIds))
			elbClient, _ := awswrapper.NewELBv2(awsKey, awsSecret, awsRegion)
			for _, lb := range lbStartedIds {
				lbResponse, err := elbClient.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
					// LoadBalancerArns: []*string{},
					LoadBalancerArns: []*string{lb},
				})

				if err != nil {
					fmt.Printf("error: %v\n", err)
				} else {
					fmt.Printf("    %v\n", *lb)
					fmt.Printf("    %v\n", lbResponse)

					_, err := elbClient.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{
						LoadBalancerArn: lb,
					})
					if err == nil {
						fmt.Printf("    LB DELETED: %v\n", *lb)
					} else {
						fmt.Printf("error: %v\n", err)
					}

				}
			}

			fmt.Printf("=== EXTRA AWS load balancers: %v\n", len(extraLBs))
			for _, lb := range extraLBs {
				fmt.Printf("    %v\n", *lb)
			}

			fmt.Printf("=== EXTRA AWS tasks: %v\n", len(extraTasks))
			for _, t := range extraTasks {
				fmt.Printf("    %v\n", *t)
			}

			wait(time.Duration(100)*time.Second, nil)
			fmt.Printf("=== EXTRA AWS security groups: %v\n", len(extraSGs))
			for _, sg := range extraSGs {
				fmt.Printf("    %v\n", *sg)

				out, err := awswrapper.DeleteSecurityGroup(awsKey, awsSecret, awsRegion, *sg)
				if err == nil {
					fmt.Printf("    %v\n", out)
				} else {
					fmt.Printf("error: %v\n", err)

					for err != nil {
						sgNode := &network.Node{}
						db.Where("role = ? and status = ?", "peer", "pending").First(&sgNode)

						if sgNode == nil {
							// db.Where("role = ? and status = ?", "peer", "running").First(&sgNode)
							// if sgNode != nil {
							// 	taskIds := keyFromConfig(sgNode.Config, "target_task_ids").([]interface{})
							// 	// fmt.Printf("task ids: %v\n", taskIds)
							// 	taskId := taskIds[0].(string)
							// 	taskStop(ecsClient, ptrTo(taskId))
							// }

							// wait(time.Duration(10)*time.Second, ptrTo(" to DELETE security groups"))
							// out, err = awswrapper.DeleteSecurityGroup(awsKey, awsSecret, awsRegion, *sg)

						} else {
							sgNode.Delete()
							wait(time.Duration(1)*time.Minute, ptrTo(" to DELETE peer node"))
							// out, err = awswrapper.DeleteSecurityGroup(awsKey, awsSecret, awsRegion, *sg)
						}
					}

					fmt.Printf("    %v\n", out)
				}
			}

			postTestNodeIds, postTestLBIds := printDBResources(db, "LEFTOVERS")
			extraNodes := extraUUIDs(preTestNodeIds, postTestNodeIds)
			extraLBIds := extraUUIDs(preTestLBIds, postTestLBIds)

			fmt.Printf("=== EXTRA DB nodes: %v\n", len(extraNodes))
			for _, node := range extraNodes {
				fmt.Printf("    %v\n", *node)
			}

			fmt.Printf("=== EXTRA DB load balancers: %v\n", len(extraLBIds))
			for _, lb := range extraLBIds {
				fmt.Printf("    %v\n", *lb)
			}
		}

	})

	Context("AWS", func() {
		// needed by peer node after hook
		var lb *network.LoadBalancer
		var lbConfig map[string]interface{}

		BeforeEach(func() {
			// FIXME: assert basic network sanity?
		})

		Context("when a single validator node is created by the master of ceremony", func() {
			ctxId := "single_validator_node"
			var validatorNode *network.Node
			var validatorScenarioCount = 36

			BeforeEach(func() {
				if validatorNode == nil {
					validatorNode, _ = createValidatorNode(ctxId, awsKey, awsSecret, awsRegion, true, &awsTasksStartedIds)

					fmt.Printf("\nvalidator node: %v\n", validatorNode)
					fmt.Printf("network : %v\n", n)
					fmt.Printf("starting stats daemon : \n")

					startStatsDaemon(n)

					lb, _ := n.LoadBalancers(db, &awsRegion, nil)
					lbIds := make([]*string, len(lb))
					for i, l := range lb {
						var lbconf map[string]interface{}
						_ = json.Unmarshal(*l.Config, &lbconf)
						lbIds[i] = ptrTo(lbconf["target_balancer_id"].(string))
					}
					// newLB := extraStrings([]*string{}, lbIds)
					newLB := extraStrings(preTestAWSLBs, lbIds)
					for _, l := range newLB {
						lbStartedIds = append(lbStartedIds, l)
					}
				}
			})

			It("should contain task id", func() {
				Expect(awsTasksStartedIds[0]).NotTo(BeNil())
				validatorScenarioCount--
			})

			It("should have status genesis", func() {
				Expect(validatorNode.Status).To(gstruct.PointTo(Equal("genesis")))
				validatorScenarioCount--
			})

			Describe("load balancing", func() {
				var loadBalancers []*network.LoadBalancer

				BeforeEach(func() {
					fmt.Printf("  NODE : %v\n", validatorNode)
					fmt.Printf("  NODE status: %v\n", *validatorNode.Status)
					loadBalancers = make([]*network.LoadBalancer, 0)
					// wait(time.Minute, nil)
					// for len(loadBalancers) == 0 {
					db.Model(&validatorNode).Association("LoadBalancers").Find(&loadBalancers)
					fmt.Printf("\n\n=== FIND load balancers: %v\n", loadBalancers)
					// }
				})

				Context("first load balancer", func() {
					var configParseError error
					var lbHost *string

					BeforeEach(func() {
						lb = loadBalancers[0]
						configParseError = json.Unmarshal(*lb.Config, &lbConfig)
						lbHost = lb.Host
					})

					It("should have one load balancer", func() {
						Expect(loadBalancers).To(HaveLen(1))
						validatorScenarioCount--
					})

					It("network and node network should match", func() {
						Expect(lb.NetworkID).To(Equal(validatorNode.NetworkID))
						validatorScenarioCount--
					})

					It("should have parseable config", func() {
						Expect(configParseError).NotTo(HaveOccurred())
						validatorScenarioCount--
					})

					It("should have correct description with target, region and name", func() {
						target := lbConfig["target_id"]
						region := lbConfig["region"].(string)
						lbDescription := fmt.Sprintf("%s - %s %s", *lb.Name, target, region)
						Expect(lb.Description).To(gstruct.PointTo(Equal(lbDescription)))
						validatorScenarioCount--
					})

					// check:
					// `websocket_url`
					// `json_rpc_url`
					// `target_task_ids`
					// `target_security_group_ids`
					// `target_balancer_id`
					// and `target_groups`
					// `load_balancer_name`
					// `load_balancer_url`

					It("should have correct websocket url", func() {
						// ws://28041de8-bc5d-4ca6-9d46-a028336-1311050691.us-west-2.elb.amazonaws.com:8051
						websocket_url := fmt.Sprintf("ws://%s:8051", *lbHost)
						Expect(lbConfig["websocket_url"]).To(Equal(websocket_url))
						validatorScenarioCount--
					})

					It("should have correct json rpc url", func() {
						// http://28041de8-bc5d-4ca6-9d46-a028336-1311050691.us-west-2.elb.amazonaws.com:8050
						json_rpc_url := fmt.Sprintf("http://%s:8050", *lbHost)
						Expect(lbConfig["json_rpc_url"]).To(Equal(json_rpc_url))
						validatorScenarioCount--
					})

					It("should have 1 task id", func() {
						// ["arn:aws:ecs:us-west-2:085843810865:task/e6eb5fef-17ef-40f1-89ef-3d2ae09f1354"]
						targetTaskIds := lbConfig["target_task_ids"]
						Expect(targetTaskIds).To(HaveLen(1))
						validatorScenarioCount--
					})

					It("should have 1 target security group", func() {
						// ["sg-027759a6c2183d446"]
						// target_security_group_ids
						targetSecurityGroupIds := lbConfig["target_security_group_ids"]
						Expect(targetSecurityGroupIds).To(HaveLen(1))
						validatorScenarioCount--
					})

					It("name and name in config should match", func() {
						// 28041de8-bc5d-4ca6-9d46-a028336
						load_balancer_name := *lb.Name
						Expect(lbConfig["load_balancer_name"]).To(Equal(load_balancer_name))

						validatorScenarioCount--
					})

					It("url and url in config should match", func() {
						// 28041de8-bc5d-4ca6-9d46-a028336-1311050691.us-west-2.elb.amazonaws.com
						load_balancer_url := *lbHost
						Expect(lbConfig["load_balancer_url"]).To(Equal(load_balancer_url))
						validatorScenarioCount--
					})

					Context("target groups", func() {
						var ports = [...]int64{5001, 8050, 8051, 8080, 30300}

						for _, p := range ports {
							cTitle := fmt.Sprintf("%v port", p)
							Context(cTitle, func() {
								var cfgTargetGroups map[string]interface{}
								var targetGroupsResp *elbv2.DescribeTargetGroupsOutput
								var targetGroupsRespError error
								var targetGroups []*elbv2.TargetGroup

								BeforeEach(func() {
									fmt.Printf("\n\nContext for %v port: ", p)

									cfgTargetGroups = lbConfig["target_groups"].(map[string]interface{})
									targetGroupName := ethcommon.Bytes2Hex(provide.Keccak256(fmt.Sprintf("%s-port-%v", lb.ID.String(), p)))[0:31]
									targetGroupsResp, targetGroupsRespError = awswrapper.GetTargetGroup(awsKey, awsSecret, "us-west-2", targetGroupName)

									targetGroups = targetGroupsResp.TargetGroups
								})

								It("should return from amazon without error", func() {
									Expect(targetGroupsRespError).NotTo(HaveOccurred())
									validatorScenarioCount--
								})

								It("should have 1 target group", func() {
									Expect(targetGroups).To(HaveLen(1))
									validatorScenarioCount--
								})

								It("should have correct port", func() {
									Expect(targetGroups[0].Port).To(gstruct.PointTo(Equal(p)))
									validatorScenarioCount--
								})

								It("should have http protocol", func() {
									Expect(targetGroups[0].Protocol).To(gstruct.PointTo(Equal("HTTP")))
									validatorScenarioCount--
								})

								It("should have correct arn", func() {
									arn := targetGroups[0].TargetGroupArn
									ctg := cfgTargetGroups[strconv.Itoa(int(p))]

									Expect(arn).To(gstruct.PointTo(Equal(ctg))) // checking lb config
									validatorScenarioCount--
								})
							})

						}

					})
				})
			})

			Context("when a peer is created and joins the network genesised by our running master of ceremony validator node", func() {
				var peerNode *network.Node
				var peerNodeConfig map[string]interface{}
				ctxId2 := "single_peer_node"
				var peerScenarioCount = 9

				BeforeEach(func() {
					if lb == nil {
						lb = &network.LoadBalancer{}
						db := dbconf.DatabaseConnection()
						db.First(&lb)
					}
					if peerNode == nil {
						peerNode, _ = createPeerNode(ctxId2, awsKey, awsSecret)

						taskIds := keyFromConfig(validatorNode.Config, "target_task_ids").([]interface{})
						fmt.Printf("    task ids: %v\n", taskIds)
						taskId := taskIds[0].(string)
						awsTasksStartedIds[1] = &taskId

						validatorNode.Reload()
						peerNode.Reload()
						err := json.Unmarshal(*peerNode.Config, &peerNodeConfig)
						Expect(err).NotTo(HaveOccurred())
					}
					fmt.Printf("=== BEFORE peerNode: %v\n", peerNode)

				})

				Describe("the peer node", func() {
					It("should have role 'peer'", func() {
						Expect(peerNode.Role).To(gstruct.PointTo(Equal("peer")))
						peerScenarioCount--
					})

					It("should have status 'running'", func() {
						Expect(peerNode.Status).To(gstruct.PointTo(Equal("running")))
						peerScenarioCount--
					})
				})

				Describe("environment variables configured by way of 'config.env'", func() {
					var envCfg map[string]interface{}
					var envCfgOk bool
					BeforeEach(func() {
						envCfg, envCfgOk = peerNodeConfig["env"].(map[string]interface{})
					})
					It("should be present", func() {
						Expect(envCfgOk).To(BeTrue())
						peerScenarioCount--
					})
					It("should contain node IP address", func() {
						prIP4 := validatorNode.PrivateIPv4
						Expect(envCfg["PEER_SET"].(string)).To(ContainSubstring(*prIP4))
						peerScenarioCount--
					})
					It("should contain 'required' literal", func() {
						Expect(envCfg["PEER_SET"].(string)).To(ContainSubstring("required"))
						peerScenarioCount--
					})
				})

				Describe("security groups", func() {
					var sg []interface{}
					var sgOk bool

					// it should have created security groups for the peer based on its security config (you are checking this already just added it for parity with my other stub comments)

					BeforeEach(func() {
						sg, sgOk = peerNodeConfig["target_security_group_ids"].([]interface{})
					})

					It("should be present", func() {
						Expect(sgOk).To(BeTrue())
						peerScenarioCount--
					})

					It("should have 1 item", func() {
						Expect(sg).To(HaveLen(1))
						peerScenarioCount--
					})
				})

				Describe("target task ids", func() {
					var taskIds []interface{}
					var taskIdsOk bool
					BeforeEach(func() {
						taskIds, taskIdsOk = peerNodeConfig["target_task_ids"].([]interface{})
					})
					It("should be present", func() {
						Expect(taskIdsOk).To(BeTrue())
						peerScenarioCount--
					})
					It("should get container", func() {
						id := taskIds[0].(string)

						containerDetails, _ := awswrapper.GetContainerDetails(awsKey, awsSecret, "us-west-2", id, nil)
						fmt.Printf("container: %#v\n", containerDetails)
						peerScenarioCount--
					})
				})

				Describe("release mutex", func() {
					It("should be succesful", func() {
						for validatorScenarioCount > 0 {
							wait(time.Duration(10)*time.Second, ptrTo(fmt.Sprintf("... %v VALIDATOR scenarios running", validatorScenarioCount)))
						}

						// time.Sleep(time.Minute)
						releaseContextMutex(ctxId2, "peer")
						Expect(contextMutexReleased(ctxId2, "peer")).To(BeTrue())
						fmt.Printf("=== CONTEXT %v MUTEX RELEASED\n", ctxId2)
					})
				})

				Context("when second peer is created", func() {

				})
				// Context("when the peer is deleted", func() {

				// 	BeforeEach(func() {

				// 		removePeerNode(ctxId2, peerNode)

				// 	})
				// 	Describe("security groups", func() {
				// 		// it should have deleted the security groups which were created for the peer when it was created
				// 	})

				// 	Describe("load balancing", func() {
				// 		Describe("target groups", func() {
				// 			// it should remove the peer from the all of the target groups it was originally added to...
				// 			// the target groups should still contain the validator
				// 			// the load balancer should still have its target group associations
				// 			// the load balancer should balance traffic to the remaining validator node
				// 		})
				// 	})

				// 	AfterEach(func() {

				// 	})
				// 	})

				AfterEach(func() {

					if !contextMutexReleased(ctxId2, "peer") {
						fmt.Printf("=== CONTEXT not yet released: %v\n", ctxId2)
						return
					}
					fmt.Printf("=== CONTEXT released: %v\n", ctxId2)

					p := 5001
					// cfgTargetGroups := lbConfig["target_groups"].(map[string]interface{})
					fmt.Printf("   AFTER peer test, lb: %v\n", lb)
					targetGroupName := ethcommon.Bytes2Hex(provide.Keccak256(fmt.Sprintf("%s-port-%v", lb.ID.String(), p)))[0:31]
					targetGroupsResp, _ := awswrapper.GetTargetGroup(awsKey, awsSecret, "us-west-2", targetGroupName)

					client, _ := awswrapper.NewELBv2(awsKey, awsSecret, awsRegion)
					targetResp, _ := client.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
						TargetGroupArn: targetGroupsResp.TargetGroups[0].TargetGroupArn,
					})
					// TargetHealthDescriptions: [{
					// 	HealthCheckPort: "5001",
					// 	Target: {
					// 	  AvailabilityZone: "us-west-2b",
					// 	  Id: "172.31.29.225",
					// 	  Port: 5001
					// 	},
					// 	TargetHealth: {
					// 	  Description: "Health checks failed with these codes: [404]",
					// 	  Reason: "Target.ResponseCodeMismatch",
					// 	  State: "unhealthy"
					// 	}
					//   },{
					// 	HealthCheckPort: "5001",
					// 	Target: {
					// 	  AvailabilityZone: "us-west-2c",
					// 	  Id: "172.31.8.99",
					// 	  Port: 5001
					// 	},
					// 	TargetHealth: {
					// 	  Description: "Health checks failed with these codes: [404]",
					// 	  Reason: "Target.ResponseCodeMismatch",
					// 	  State: "unhealthy"
					// 	}
					//   }]

					fmt.Printf("    response: %v\n", targetResp)
					// targetGroups := targetGroupsResp.TargetGroups
					Expect(targetResp.TargetHealthDescriptions).To(HaveLen(2))

					fmt.Printf("=== AFTER peer node: %v\n", peerNode)
					_ = removeNode(ctxId2, peerNode)

					status, _ := n.Status(false)
					currentBlockNumber := status.Block

					fmt.Printf("\n\n=== AFTER peer test, current block: %v\n", currentBlockNumber)

					// currentNetworkStats :=
					// blockNumber
					wait(time.Duration(20)*time.Second, nil)

					status, _ = n.Status(false)
					newBlockNumber := status.Block

					fmt.Printf("\n\n=== AFTER peer test, ew block: %v\n", newBlockNumber)

					Expect(newBlockNumber).To(BeNumerically(">", currentBlockNumber))

					// check target groups number

					wait(time.Duration(4)*time.Minute, ptrTo("START target group removal"))
					for len(targetResp.TargetHealthDescriptions) > 1 {
						wait(time.Duration(10)*time.Second, nil)
						targetResp, _ = client.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
							TargetGroupArn: targetGroupsResp.TargetGroups[0].TargetGroupArn,
						})
						fmt.Printf("  targets: %v\n", targetResp)
					}

					Expect(len(targetResp.TargetHealthDescriptions)).To(Equal(1))

				})
			})
			// }

			Describe("release mutex", func() {
				It("should be succesful", func() {
					// time.Sleep(time.Minute)
					releaseContextMutex(ctxId, "validator")
					Expect(contextMutexReleased(ctxId, "validator")).To(BeTrue())
					fmt.Printf("=== CONTEXT %v MUTEX RELEASED\n", ctxId)
				})
			})

			AfterEach(func() {
				if !contextMutexReleased(ctxId, "validator") {
					fmt.Printf("=== CONTEXT not yet released: %v\n", ctxId)
					return
				}
				fmt.Printf("=== CONTEXT released: %v\n", ctxId)

				removeNode(ctxId, validatorNode)

				// lb := &network.LoadBalancer{}
				// db.Model(&lb).First(&lb)
				// fmt.Printf("lb: %#v\n", lb)
				// lb.Deprovision(db)
				wait(time.Duration(2)*time.Minute, nil)
				loadBalancers := make([]*network.LoadBalancer, 0)
				db.Model(&validatorNode).Association("LoadBalancers").Find(&loadBalancers)
				Expect(loadBalancers).To(HaveLen(0))
			})

		})
	})

	Context("create node from UI", func() {
		// cURL request in node_curl.txt file
		// auth token should be taken from Rails app Provide
		// spec.json should have the last account from ng app with identities list
		// Thus, to test node creation turn-around we need to:
		//   1. create or get identity from ng app
		//   2. update spec.json
		//   3. upload to dropbox
		//   4. use it in curl request
		//   5. create a user session
		//   6. get auth token
		//   7. send curl with token
		//   8. wait couple of minutes
	})

	// FContext("create node from json", func() {
	// 	It("should decode json", func() {
	// 		awsKey := os.Getenv("AWS_KEY")
	// 		awsSecret := os.Getenv("AWS_SECRET")

	// 		node := &network.Node{}
	// 		nodeConfig := fmt.Sprintf(`{"config":{"credentials":{"aws_access_key_id":"%s","aws_secret_access_key":"%s"},"engine_id":"aura","engines":"[{\"id\":\"aura\",\"name\":\"Authority Round\",\"enabled\":true},{\"id\":\"clique\",\"name\":\"Clique\",\"enabled\":true}]","env":{"CHAIN_SPEC_URL":"https://www.dropbox.com/s/515w1hxayztx80e/spec.json?dl=1","ENGINE_SIGNER":"0x1e23ce07ebC8d1f515C5f04101018e8A9Ab9353f","NETWORK_ID":"1537367446","TRACING":"on","PRUNING":"archive","FAT_DB":"on","LOGGING":"verbose","CHAIN":"mk"},"protocol_id":"poa","provider_id":"docker","providers":"[{\"id\":\"docker\",\"name\":\"Docker\",\"img_src_dark\":\"https://s3.amazonaws.com/provide.services/img/docker-dark.png\",\"img_src_light\":\"https://s3.amazonaws.com/provide.services/img/docker-light.png\",\"enabled\":true,\"img_src\":\"/assets/docker-c9b76b9cdf162dd88c2c685c5f2fa45deea5209de420b220f2d35273f33a397a.png\"}]","rc.d":"{\n  \"CHAIN_SPEC_URL\": \"https://www.dropbox.com/s/xuihpmm72o5m9ir/spec.json?dl=1\",\n  \"ENGINE_SIGNER\": \"0x41f4b57D711c93a22F0Ac580a842eac52E4bC505\",\n  \"NETWORK_ID\": \"1537367446\",\n  \"TRACING\": \"on\",\n  \"PRUNING\": \"archive\",\n  \"FAT_DB\": \"on\",\n  \"LOGGING\": \"verbose\",\n  \"CHAIN\": \"kt\"\n}","region":"us-west-2","role":"peer","roles":"[{\"id\":\"peer\",\"name\":\"Peer\",\"config\":{\"allows_multiple_deployment\":true,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"JSON_RPC_URL\\\": null, \\\"NETWORK_ID\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":2},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"validator\",\"name\":\"Validator\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"CHAIN_SPEC_URL\\\": null, \\\"ENGINE_SIGNER\\\": null, \\\"NETWORK_ID\\\": null, \\\"ENGINE_SIGNER_PRIVATE_KEY\\\": null}\"},\"ubuntu-vm\":{\"provide.network\":\"#!/bin/bash\\n\\nservice provide.network stop\\nrm -rf /opt/provide.network\\n\\nwget -d --output-document=/opt/spec.json --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ chainspecUrl }}\\n\\nwget -d --output-document=/opt/bootnodes.txt --header=\\\"x-api-authorization: Basic {{ apiAuthorization }}\\\" {{ bootnodesUrl }}\\n\\nservice provide.network start\\n\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"ubuntu-vm\",\"docker\"]},{\"id\":\"explorer\",\"name\":\"Block Explorer\",\"config\":{\"allows_multiple_deployment\":false,\"default_rcd\":{\"docker\":{\"provide.network\":\"{\\\"JSON_RPC_URL\\\": null}\"}},\"quickclone_recommended_node_count\":1},\"supported_provider_ids\":[\"docker\"]}]","target_id":"aws"},"network_id":"36a5f8e0-bfc1-49f8-a7ba-86457bb52912"}`, awsKey, awsSecret)

	// 		err := json.Unmarshal([]byte(nodeConfig), node)
	// 		// if err != nil {
	// 		// 	fmt.Printf("error: %v\n", err)
	// 		// }
	// 		Expect(err).NotTo(HaveOccurred())

	// 		fmt.Printf("%#v", node)
	// 		Expect(node.Role).To(Equal("peer"))

	// 	})
	// })

})
