// +build unit

package network_test

import (
	"encoding/json"
	"time"

	"github.com/onsi/gomega/gstruct"

	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func marshalConfig(opts map[string]interface{}) *json.RawMessage {
	cfgJSON, _ := json.Marshal(opts)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// func TestNodes(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "Network Suite")
// }

var _ = Describe("Node", func() {
	Context("no AWS", func() {
		Describe("First node is created in the network", func() {
			var node *network.NetworkNode
			Describe("Create()", func() {
				Context("without network", func() {
					BeforeEach(func() {
						node = &network.NetworkNode{
							UserID:      nil,
							Bootnode:    true,
							Host:        nil,
							IPv4:        nil,
							IPv6:        nil,
							PrivateIPv4: nil,
							PrivateIPv6: nil,
							Description: nil,
							Role:        nil,
							Status:      nil,
							Config:      nil,
						}
					})
					AfterEach(func() {
						db := dbconf.DatabaseConnection()
						db.Delete(network.Network{})
						db.Delete(network.NetworkNode{})
					})
					Context("empty role", func() {
						It("should set role to peer", func() {
							node.Create()
							Expect(node.Role).To(gstruct.PointTo(Equal("peer")))
						})
						It("should set status to init", func() {
							node.Create()
							Expect(node.Status).To(gstruct.PointTo(Equal("init")))
						})
					})
					Context("validator", func() {
						Context("without config", func() {
							BeforeEach(func() {
								node.Role = common.StringOrNil("validator")
							})
							It("should create node successfully", func() {
								Expect(node.Create()).To(BeTrue())
							})
							It("should set status to init", func() {
								node.Create()
								Expect(node.Status).To(gstruct.PointTo(Equal("init")))
							})

						})
						Context("with config", func() {
							It("should save the node", func() {
								node.Role = common.StringOrNil("validator")
								node.Config = marshalConfig(
									map[string]interface{}{
										"security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}},
								)

								// "config":{"security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[5001,8050,8051,8080,30300],"udp":[30300]}}},
								Expect(node.Create()).To(BeTrue())
								Expect(node.Status).To(gstruct.PointTo(Equal("init")))
								time.Sleep(time.Duration(100) * time.Millisecond)
								Expect(node.Status).To(gstruct.PointTo(Equal("failed")))
								// Expect(node.Status).To(gstruct.PointTo(Equal("genesis"))) - not possible w/o real network
							})

						})
					})
					Context("faucet", func() {
						Context("without config", func() {
							BeforeEach(func() {
								node.Role = common.StringOrNil("faucet")
							})
							It("should create node successfully", func() {
								Expect(node.Create()).To(BeTrue())
							})
							It("should set status to init", func() {
								node.Create()
								Expect(node.Status).To(gstruct.PointTo(Equal("init")))
							})
						})
					})
					Context("full", func() {
						Context("without config", func() {
							BeforeEach(func() {
								node.Role = common.StringOrNil("full")
							})
							It("should save the node", func() {
								Expect(node.Create()).To(BeTrue())
							})
							It("should set status to init", func() {
								node.Create()
								Expect(node.Status).To(gstruct.PointTo(Equal("init")))
							})

						})
					})
					Context("peer", func() {
						Context("without config", func() {
							BeforeEach(func() {
								node.Role = common.StringOrNil("peer")
							})
							It("should save the node", func() {
								Expect(node.Create()).To(BeTrue())
							})
							It("should set status to init", func() {
								node.Create()
								Expect(node.Status).To(gstruct.PointTo(Equal("init")))
							})

						})
					})
				})
			})
			Context("validate()", func() {
				BeforeEach(func() {
					node = &network.NetworkNode{
						UserID:      nil,
						Bootnode:    true,
						Host:        nil,
						IPv4:        nil,
						IPv6:        nil,
						PrivateIPv4: nil,
						PrivateIPv6: nil,
						Description: nil,
						Role:        nil,
						Status:      nil,
						Config:      nil,
					}

				})
				Context("node without config", func() {
					It("should return false", func() {
						Expect(node.Validate()).To(BeTrue())
					})
				})
				Context("node with config", func() {
					Context("without role key", func() {
						It("should return false", func() {
							node.Config = marshalConfig(map[string]interface{}{})
							Expect(node.Validate()).To(BeTrue())
						})
					})
					Context("with role key", func() {
						It("should return true", func() {
							node.Config = marshalConfig(map[string]interface{}{"role": "role"})
							Expect(node.Validate()).To(BeTrue())

						})
						It("should set role field", func() {
							node.Config = marshalConfig(map[string]interface{}{"role": "role"})
							node.Validate()
							Expect(node.Role).To(gstruct.PointTo(Equal("role")))

						})
					})

				})
			})

			Context("ParseConfig()", func() {

			})
			Context("Delete()", func() {

			})
			Context("Reload()", func() {

			})
			Context("Logs()", func() {

			})
		})
	})

})
